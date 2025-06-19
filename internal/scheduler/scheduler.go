package scheduler

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"ims/internal/domain"
	"ims/internal/service"
)

type Scheduler struct {
	service      *service.MessageService
	auditService service.AuditService
	interval     time.Duration
	batchSize    int

	mu        sync.Mutex
	ticker    *time.Ticker
	done      chan struct{}
	running   int32
	startedAt *time.Time
}

func NewScheduler(service *service.MessageService, auditService service.AuditService, interval time.Duration, batchSize int) *Scheduler {
	return &Scheduler{
		service:      service,
		auditService: auditService,
		interval:     interval,
		batchSize:    batchSize,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 1 {
		return domain.ErrSchedulerRunning
	}

	s.ticker = time.NewTicker(s.interval)
	s.done = make(chan struct{})
	now := time.Now()
	s.startedAt = &now

	atomic.StoreInt32(&s.running, 1)

	// Log scheduler started event
	if s.auditService != nil {
		go func() {
			if err := s.auditService.LogSchedulerStarted(context.Background()); err != nil {
				log.Printf("Failed to log scheduler started event: %v", err)
			}
		}()
	}

	// Use background context for scheduler operations, not the HTTP request context
	schedulerCtx := context.Background()
	go s.run(schedulerCtx)

	// Process immediately on start
	go func() {
		s.processBatch(schedulerCtx)
	}()

	log.Printf("Scheduler started with interval: %v, batch size: %d", s.interval, s.batchSize)
	return nil
}

func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.running) == 0 {
		return domain.ErrSchedulerNotRunning
	}

	close(s.done)
	s.ticker.Stop()
	atomic.StoreInt32(&s.running, 0)
	s.startedAt = nil

	// Log scheduler stopped event
	if s.auditService != nil {
		go func() {
			if err := s.auditService.LogSchedulerStopped(context.Background()); err != nil {
				log.Printf("Failed to log scheduler stopped event: %v", err)
			}
		}()
	}

	log.Println("Scheduler stopped")
	return nil
}

func (s *Scheduler) IsRunning() bool {
	return atomic.LoadInt32(&s.running) == 1
}

func (s *Scheduler) GetStatus() (bool, *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.IsRunning(), s.startedAt
}

func (s *Scheduler) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopping due to context cancellation")
			s.Stop()
			return
		case <-s.done:
			log.Println("Scheduler stopping due to done signal")
			return
		case <-s.ticker.C:
			s.processBatch(ctx)
		}
	}
}

func (s *Scheduler) processBatch(ctx context.Context) {
	// Create a unique batch ID for tracking
	batchID := uuid.New()
	startTime := time.Now()

	// Create a timeout context for batch processing (use a reasonable timeout)
	timeout := 30 * time.Second
	if s.interval > time.Minute {
		// Use up to half the interval for batch processing, but at least 30 seconds
		timeout = s.interval / 2
	}

	batchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("Processing batch %s of %d messages", batchID.String(), s.batchSize)

	// Log batch started
	if s.auditService != nil {
		go func() {
			if err := s.auditService.LogBatchStarted(context.Background(), batchID, s.batchSize); err != nil {
				log.Printf("Failed to log batch started event: %v", err)
			}
		}()
	}

	// Process the batch
	err := s.service.ProcessMessages(batchCtx, s.batchSize)
	duration := time.Since(startTime)

	// Log batch completion or failure
	if s.auditService != nil {
		go func() {
			if err != nil {
				if logErr := s.auditService.LogBatchFailed(context.Background(), batchID, duration, err); logErr != nil {
					log.Printf("Failed to log batch failed event: %v", logErr)
				}
			} else {
				// For now, we'll log with generic success count since we don't have detailed metrics from ProcessMessages
				// In a real implementation, ProcessMessages would return success/failure counts
				if logErr := s.auditService.LogBatchCompleted(context.Background(), batchID, duration, s.batchSize, 0); logErr != nil {
					log.Printf("Failed to log batch completed event: %v", logErr)
				}
			}
		}()
	}

	if err != nil {
		log.Printf("Error processing messages batch %s: %v", batchID.String(), err)
	} else {
		log.Printf("Completed processing batch %s in %v", batchID.String(), duration)
	}
}
