// Package scheduler - Queue-based scheduler implementation
package scheduler

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"ims/internal/domain"
	"ims/internal/queue"
	"ims/internal/service"
)

// QueueScheduler handles message processing using queue abstraction
type QueueScheduler struct {
	queueManager queue.QueueManager
	messageQueue queue.MessageQueue
	webhook      *service.WebhookClient
	auditService service.AuditService
	maxLength    int

	mu        sync.Mutex
	done      chan struct{}
	running   int32
	startedAt *time.Time
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewQueueScheduler creates a new queue-based scheduler
func NewQueueScheduler(
	queueManager queue.QueueManager,
	webhook *service.WebhookClient,
	auditService service.AuditService,
	maxLength int,
) *QueueScheduler {
	return &QueueScheduler{
		queueManager: queueManager,
		messageQueue: queueManager.GetQueue(),
		webhook:      webhook,
		auditService: auditService,
		maxLength:    maxLength,
	}
}

// Start starts the queue-based scheduler
func (qs *QueueScheduler) Start(ctx context.Context) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if atomic.LoadInt32(&qs.running) == 1 {
		return domain.ErrSchedulerRunning
	}

	qs.done = make(chan struct{})
	now := time.Now()
	qs.startedAt = &now
	qs.ctx, qs.cancel = context.WithCancel(ctx)

	atomic.StoreInt32(&qs.running, 1)

	// Log scheduler started event
	if qs.auditService != nil {
		go func() {
			if err := qs.auditService.LogSchedulerStarted(context.Background()); err != nil {
				log.Printf("Failed to log scheduler started event: %v", err)
			}
		}()
	}

	queueType := qs.messageQueue.GetQueueType()
	log.Printf("Queue scheduler started using %s queue", queueType)

	// Start consuming messages
	go qs.consume()

	return nil
}

// Stop stops the queue-based scheduler
func (qs *QueueScheduler) Stop() error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if atomic.LoadInt32(&qs.running) == 0 {
		return domain.ErrSchedulerNotRunning
	}

	// Signal shutdown
	if qs.cancel != nil {
		qs.cancel()
	}

	close(qs.done)
	atomic.StoreInt32(&qs.running, 0)
	qs.startedAt = nil

	// Close queue connection
	if err := qs.messageQueue.Close(); err != nil {
		log.Printf("Error closing message queue: %v", err)
	}

	// Log scheduler stopped event
	if qs.auditService != nil {
		go func() {
			if err := qs.auditService.LogSchedulerStopped(context.Background()); err != nil {
				log.Printf("Failed to log scheduler stopped event: %v", err)
			}
		}()
	}

	log.Println("Queue scheduler stopped")
	return nil
}

// IsRunning returns whether the scheduler is running
func (qs *QueueScheduler) IsRunning() bool {
	return atomic.LoadInt32(&qs.running) == 1
}

// GetStatus returns the scheduler status and start time
func (qs *QueueScheduler) GetStatus() (bool, *time.Time) {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	return qs.IsRunning(), qs.startedAt
}

// consume starts consuming messages from the queue
func (qs *QueueScheduler) consume() {
	handler := func(ctx context.Context, message *domain.Message) error {
		return qs.processMessage(ctx, message)
	}

	if err := qs.messageQueue.Consume(qs.ctx, handler); err != nil {
		if qs.ctx.Err() != nil {
			log.Println("Queue consumption stopped due to context cancellation")
		} else {
			log.Printf("Error consuming messages from queue: %v", err)
		}
	}
}

// processMessage processes a single message
func (qs *QueueScheduler) processMessage(ctx context.Context, msg *domain.Message) error {
	batchID := uuid.New() // Create a batch ID for this single message
	startTime := time.Now()

	log.Printf("Processing message %s to %s (batch %s)", msg.ID, msg.PhoneNumber, batchID)

	// Log batch started (for single message)
	if qs.auditService != nil {
		go func() {
			if err := qs.auditService.LogBatchStarted(context.Background(), batchID, 1); err != nil {
				log.Printf("Failed to log batch started event: %v", err)
			}
		}()
	}

	// Validate message content length
	if len(msg.Content) > qs.maxLength {
		err := domain.ErrMessageTooLong
		log.Printf("Message %s exceeds maximum length (%d > %d)", msg.ID, len(msg.Content), qs.maxLength)
		qs.logBatchResult(batchID, startTime, err, 0, 1)
		return err
	}

	// Create timeout context for webhook call
	webhookCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	log.Printf("Sending message %s to %s", msg.ID, msg.PhoneNumber)

	// Send via webhook
	resp, err := qs.webhook.Send(webhookCtx, msg.PhoneNumber, msg.Content)

	if err != nil {
		log.Printf("Failed to send message %s: %v", msg.ID, err)
		qs.logBatchResult(batchID, startTime, err, 0, 1)
		return err
	}

	log.Printf("Message %s sent successfully, webhook response ID: %s", msg.ID, resp.MessageID)
	qs.logBatchResult(batchID, startTime, nil, 1, 0)

	return nil
}

// logBatchResult logs the batch processing result
func (qs *QueueScheduler) logBatchResult(batchID uuid.UUID, startTime time.Time, err error, successCount, failureCount int) {
	duration := time.Since(startTime)

	if qs.auditService != nil {
		go func() {
			if err != nil {
				if logErr := qs.auditService.LogBatchFailed(context.Background(), batchID, duration, err); logErr != nil {
					log.Printf("Failed to log batch failed event: %v", logErr)
				}
			} else {
				if logErr := qs.auditService.LogBatchCompleted(context.Background(), batchID, duration, successCount, failureCount); logErr != nil {
					log.Printf("Failed to log batch completed event: %v", logErr)
				}
			}
		}()
	}
}
