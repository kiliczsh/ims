// Package queue - Queue manager implementation
package queue

import (
	"fmt"

	"ims/internal/config"
	"ims/internal/repository"
)

// Manager implements QueueManager interface
type Manager struct {
	config      *config.Config
	queue       MessageQueue
	messageRepo repository.MessageRepository
}

// NewManager creates a new queue manager
func NewManager(cfg *config.Config, messageRepo repository.MessageRepository) (*Manager, error) {
	manager := &Manager{
		config:      cfg,
		messageRepo: messageRepo,
	}

	// Initialize the appropriate queue implementation
	if cfg.RabbitMQ.Enabled && cfg.RabbitMQ.URL != "" {
		// Initialize RabbitMQ queue
		rabbitQueue, err := NewRabbitMQQueue(cfg.RabbitMQ)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RabbitMQ queue: %w", err)
		}
		manager.queue = rabbitQueue
	} else {
		// Initialize database queue (default)
		dbQueue := NewDatabaseQueue(
			messageRepo,
			cfg.Scheduler.BatchSize,
			cfg.Scheduler.Interval,
		)
		manager.queue = dbQueue
	}

	return manager, nil
}

// GetQueue returns the appropriate queue implementation
func (m *Manager) GetQueue() MessageQueue {
	return m.queue
}

// IsRabbitMQEnabled returns true if RabbitMQ is enabled and configured
func (m *Manager) IsRabbitMQEnabled() bool {
	return m.config.RabbitMQ.Enabled && m.config.RabbitMQ.URL != ""
}

// Close closes the queue manager and its underlying queue
func (m *Manager) Close() error {
	if m.queue != nil {
		return m.queue.Close()
	}
	return nil
}

// GetQueueType returns the type of the current queue
func (m *Manager) GetQueueType() QueueType {
	if m.queue != nil {
		return m.queue.GetQueueType()
	}
	return QueueTypeDatabase // default fallback
}
