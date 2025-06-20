// Package queue provides an abstraction layer for message queuing
// supporting both database polling and RabbitMQ implementations
package queue

import (
	"context"

	"ims/internal/domain"
)

// MessageQueue defines the interface for message queue operations
type MessageQueue interface {
	// Publish publishes a message to the queue
	Publish(ctx context.Context, message *domain.Message) error

	// Consume starts consuming messages from the queue
	// The handler function is called for each message received
	Consume(ctx context.Context, handler MessageHandler) error

	// Close closes the queue connection
	Close() error

	// GetQueueType returns the type of queue implementation
	GetQueueType() QueueType
}

// MessageHandler defines the function signature for handling consumed messages
type MessageHandler func(ctx context.Context, message *domain.Message) error

// QueueType represents the type of queue implementation
type QueueType string

const (
	QueueTypeDatabase QueueType = "database"
	QueueTypeRabbitMQ QueueType = "rabbitmq"
)

// QueueManager manages different queue implementations
type QueueManager interface {
	// GetQueue returns the appropriate queue implementation based on configuration
	GetQueue() MessageQueue

	// IsRabbitMQEnabled returns true if RabbitMQ is enabled and configured
	IsRabbitMQEnabled() bool
}
