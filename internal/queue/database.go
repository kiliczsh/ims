// Package queue - Database queue implementation
package queue

import (
	"context"
	"fmt"
	"time"

	"ims/internal/domain"
	"ims/internal/repository"
)

// DatabaseQueue implements MessageQueue using database polling
type DatabaseQueue struct {
	repo      repository.MessageRepository
	batchSize int
	interval  time.Duration
}

// NewDatabaseQueue creates a new database queue implementation
func NewDatabaseQueue(repo repository.MessageRepository, batchSize int, interval time.Duration) *DatabaseQueue {
	return &DatabaseQueue{
		repo:      repo,
		batchSize: batchSize,
		interval:  interval,
	}
}

// Publish publishes a message to the database
func (dq *DatabaseQueue) Publish(ctx context.Context, message *domain.Message) error {
	return dq.repo.CreateMessage(ctx, message)
}

// Consume starts consuming messages from the database using polling
func (dq *DatabaseQueue) Consume(ctx context.Context, handler MessageHandler) error {
	ticker := time.NewTicker(dq.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := dq.processMessages(ctx, handler); err != nil {
				// Log error but continue processing
				fmt.Printf("Error processing messages: %v\n", err)
			}
		}
	}
}

// processMessages processes a batch of messages from the database
func (dq *DatabaseQueue) processMessages(ctx context.Context, handler MessageHandler) error {
	// Process unsent messages first
	if err := dq.processBatch(ctx, handler, dq.getUnsentMessages); err != nil {
		return fmt.Errorf("failed to process unsent messages: %w", err)
	}

	// Process retryable messages
	if err := dq.processBatch(ctx, handler, dq.getRetryableMessages); err != nil {
		return fmt.Errorf("failed to process retryable messages: %w", err)
	}

	return nil
}

// processBatch processes a batch of messages using the provided getter function
func (dq *DatabaseQueue) processBatch(ctx context.Context, handler MessageHandler, getter func(context.Context, int) ([]*domain.Message, error)) error {
	messages, err := getter(ctx, dq.batchSize)
	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := handler(ctx, message); err != nil {
			return fmt.Errorf("failed to handle message %s: %w", message.ID, err)
		}
	}

	return nil
}

// getUnsentMessages gets unsent messages from the repository
func (dq *DatabaseQueue) getUnsentMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	return dq.repo.GetUnsentMessages(ctx, limit)
}

// getRetryableMessages gets retryable messages from the repository
func (dq *DatabaseQueue) getRetryableMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	return dq.repo.GetRetryableMessages(ctx, limit)
}

// Close closes the database queue (no-op for database implementation)
func (dq *DatabaseQueue) Close() error {
	return nil
}

// GetQueueType returns the queue type
func (dq *DatabaseQueue) GetQueueType() QueueType {
	return QueueTypeDatabase
}
