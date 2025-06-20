// Package queue - RabbitMQ queue implementation
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ims/internal/config"
	"ims/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQQueue implements MessageQueue using RabbitMQ
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.RabbitMQConfig
}

// NewRabbitMQQueue creates a new RabbitMQ queue implementation
func NewRabbitMQQueue(cfg config.RabbitMQConfig) (*RabbitMQQueue, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rq := &RabbitMQQueue{
		conn:    conn,
		channel: channel,
		config:  cfg,
	}

	// Declare queues
	if err := rq.declareQueues(); err != nil {
		rq.Close()
		return nil, fmt.Errorf("failed to declare queues: %w", err)
	}

	return rq, nil
}

// declareQueues declares all required queues
func (rq *RabbitMQQueue) declareQueues() error {
	queues := []string{
		rq.config.MessagesQueue,
		rq.config.RetryQueue,
		rq.config.DeadLetterQueue,
	}

	for _, queueName := range queues {
		_, err := rq.channel.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}
	}

	return nil
}

// Publish publishes a message to RabbitMQ
func (rq *RabbitMQQueue) Publish(ctx context.Context, message *domain.Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = rq.channel.Publish(
		"",                      // exchange
		rq.config.MessagesQueue, // routing key
		false,                   // mandatory
		false,                   // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // make message persistent
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Consume starts consuming messages from RabbitMQ
func (rq *RabbitMQQueue) Consume(ctx context.Context, handler MessageHandler) error {
	// Consume from main messages queue
	go rq.consumeFromQueue(ctx, rq.config.MessagesQueue, handler)

	// Consume from retry queue
	go rq.consumeFromQueue(ctx, rq.config.RetryQueue, handler)

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

// consumeFromQueue consumes messages from a specific queue
func (rq *RabbitMQQueue) consumeFromQueue(ctx context.Context, queueName string, handler MessageHandler) error {
	msgs, err := rq.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack (we'll ack manually)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer for queue %s: %w", queueName, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery := <-msgs:
			if delivery.Body == nil {
				continue
			}

			var message domain.Message
			if err := json.Unmarshal(delivery.Body, &message); err != nil {
				fmt.Printf("Failed to unmarshal message: %v\n", err)
				delivery.Nack(false, false) // reject and don't requeue
				continue
			}

			if err := handler(ctx, &message); err != nil {
				fmt.Printf("Failed to handle message %s: %v\n", message.ID, err)
				// Handle retry logic
				rq.handleRetry(ctx, &message, delivery, err)
			} else {
				delivery.Ack(false) // acknowledge successful processing
			}
		}
	}
}

// handleRetry handles message retry logic
func (rq *RabbitMQQueue) handleRetry(ctx context.Context, message *domain.Message, delivery amqp.Delivery, handlerErr error) {
	retryCount := rq.getRetryCount(delivery.Headers)
	retryCount++

	if retryCount > rq.config.MaxRetries {
		// Move to dead letter queue
		rq.moveToDeadLetterQueue(ctx, message, fmt.Sprintf("Max retries exceeded: %v", handlerErr))
		delivery.Ack(false) // acknowledge to remove from current queue
		return
	}

	// Calculate retry delay
	delay := time.Duration(retryCount*retryCount*rq.config.RetryDelayMultiplier) * time.Second

	// Publish to retry queue with delay
	go func() {
		time.Sleep(delay)
		rq.publishToRetryQueue(ctx, message, retryCount)
	}()

	delivery.Ack(false) // acknowledge to remove from current queue
}

// getRetryCount extracts retry count from message headers
func (rq *RabbitMQQueue) getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	if count, ok := headers["retry_count"].(int); ok {
		return count
	}

	return 0
}

// publishToRetryQueue publishes a message to the retry queue
func (rq *RabbitMQQueue) publishToRetryQueue(ctx context.Context, message *domain.Message, retryCount int) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message for retry: %w", err)
	}

	headers := amqp.Table{
		"retry_count": retryCount,
	}

	err = rq.channel.Publish(
		"",                   // exchange
		rq.config.RetryQueue, // routing key
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message to retry queue: %w", err)
	}

	return nil
}

// moveToDeadLetterQueue moves a message to the dead letter queue
func (rq *RabbitMQQueue) moveToDeadLetterQueue(ctx context.Context, message *domain.Message, reason string) error {
	dlqMessage := map[string]interface{}{
		"original_message": message,
		"failure_reason":   reason,
		"moved_at":         time.Now(),
	}

	body, err := json.Marshal(dlqMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	err = rq.channel.Publish(
		"",                        // exchange
		rq.config.DeadLetterQueue, // routing key
		false,                     // mandatory
		false,                     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message to dead letter queue: %w", err)
	}

	return nil
}

// Close closes the RabbitMQ connection
func (rq *RabbitMQQueue) Close() error {
	if rq.channel != nil {
		if err := rq.channel.Close(); err != nil {
			fmt.Printf("Error closing RabbitMQ channel: %v\n", err)
		}
	}

	if rq.conn != nil {
		if err := rq.conn.Close(); err != nil {
			fmt.Printf("Error closing RabbitMQ connection: %v\n", err)
		}
	}

	return nil
}

// GetQueueType returns the queue type
func (rq *RabbitMQQueue) GetQueueType() QueueType {
	return QueueTypeRabbitMQ
}
