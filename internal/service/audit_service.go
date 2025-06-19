package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"ims/internal/domain"
	"ims/internal/repository"
)

type AuditService interface {
	// Batch-related audit logging
	LogBatchStarted(ctx context.Context, batchID uuid.UUID, messageCount int) error
	LogBatchCompleted(ctx context.Context, batchID uuid.UUID, duration time.Duration, successCount, failureCount int) error
	LogBatchFailed(ctx context.Context, batchID uuid.UUID, duration time.Duration, err error) error

	// Message-related audit logging
	LogMessageSent(ctx context.Context, messageID uuid.UUID, duration time.Duration, webhookURL string) error
	LogMessageFailed(ctx context.Context, messageID uuid.UUID, duration time.Duration, webhookURL string, err error) error

	// Webhook-related audit logging
	LogWebhookRequest(ctx context.Context, messageID uuid.UUID, webhookURL, method string, requestBody interface{}) error
	LogWebhookResponse(ctx context.Context, messageID uuid.UUID, webhookURL string, statusCode int, duration time.Duration, responseBody interface{}) error

	// API request audit logging
	LogAPIRequest(ctx context.Context, requestID, method, endpoint string, statusCode int, duration time.Duration, userAgent string) error

	// Scheduler audit logging
	LogSchedulerStarted(ctx context.Context) error
	LogSchedulerStopped(ctx context.Context) error

	// Generic audit logging
	Log(ctx context.Context, auditLog *domain.AuditLog) error

	// Query audit logs
	GetAuditLogs(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error)
	GetBatchAuditLogs(ctx context.Context, batchID string) ([]*domain.AuditLog, error)
	GetMessageAuditLogs(ctx context.Context, messageID string) ([]*domain.AuditLog, error)
	GetAuditLogStats(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error)

	// Maintenance
	CleanupOldAuditLogs(ctx context.Context, days int) (int64, error)
}

type auditService struct {
	auditRepo repository.AuditRepository
}

func NewAuditService(auditRepo repository.AuditRepository) AuditService {
	return &auditService{
		auditRepo: auditRepo,
	}
}

func (s *auditService) LogBatchStarted(ctx context.Context, batchID uuid.UUID, messageCount int) error {
	auditLog := domain.NewAuditLog(domain.EventBatchStarted, "Batch Processing Started").
		WithDescription(fmt.Sprintf("Started processing batch with %d messages", messageCount)).
		WithBatchID(batchID).
		WithMessageCounts(messageCount, 0, 0).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogBatchCompleted(ctx context.Context, batchID uuid.UUID, duration time.Duration, successCount, failureCount int) error {
	totalCount := successCount + failureCount
	auditLog := domain.NewAuditLog(domain.EventBatchCompleted, "Batch Processing Completed").
		WithDescription(fmt.Sprintf("Completed processing batch - %d successful, %d failed", successCount, failureCount)).
		WithBatchID(batchID).
		WithDuration(duration).
		WithMessageCounts(totalCount, successCount, failureCount).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogBatchFailed(ctx context.Context, batchID uuid.UUID, duration time.Duration, err error) error {
	auditLog := domain.NewAuditLog(domain.EventBatchFailed, "Batch Processing Failed").
		WithDescription(fmt.Sprintf("Batch processing failed: %s", err.Error())).
		WithBatchID(batchID).
		WithDuration(duration).
		WithMetadata("error", err.Error()).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogMessageSent(ctx context.Context, messageID uuid.UUID, duration time.Duration, webhookURL string) error {
	auditLog := domain.NewAuditLog(domain.EventMessageSent, "Message Sent Successfully").
		WithDescription("Message sent to webhook successfully").
		WithMessageID(messageID).
		WithDuration(duration).
		WithMetadata("webhook_url", webhookURL).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogMessageFailed(ctx context.Context, messageID uuid.UUID, duration time.Duration, webhookURL string, err error) error {
	auditLog := domain.NewAuditLog(domain.EventMessageFailed, "Message Send Failed").
		WithDescription(fmt.Sprintf("Failed to send message: %s", err.Error())).
		WithMessageID(messageID).
		WithDuration(duration).
		WithMetadata("webhook_url", webhookURL).
		WithMetadata("error", err.Error()).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogWebhookRequest(ctx context.Context, messageID uuid.UUID, webhookURL, method string, requestBody interface{}) error {
	auditLog := domain.NewAuditLog(domain.EventWebhookRequest, "Webhook Request Sent").
		WithDescription("Sent request to webhook endpoint").
		WithMessageID(messageID).
		WithHTTPDetails(method, webhookURL, 0).
		WithMetadata("request_body", requestBody).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogWebhookResponse(ctx context.Context, messageID uuid.UUID, webhookURL string, statusCode int, duration time.Duration, responseBody interface{}) error {
	auditLog := domain.NewAuditLog(domain.EventWebhookResponse, "Webhook Response Received").
		WithDescription(fmt.Sprintf("Received response from webhook with status %d", statusCode)).
		WithMessageID(messageID).
		WithHTTPDetails("POST", webhookURL, statusCode).
		WithDuration(duration).
		WithMetadata("response_body", responseBody).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogAPIRequest(ctx context.Context, requestID, method, endpoint string, statusCode int, duration time.Duration, userAgent string) error {
	auditLog := domain.NewAuditLog(domain.EventAPIRequest, "API Request Processed").
		WithDescription(fmt.Sprintf("Processed %s request to %s", method, endpoint)).
		WithRequestID(requestID).
		WithHTTPDetails(method, endpoint, statusCode).
		WithDuration(duration).
		WithMetadata("user_agent", userAgent).
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogSchedulerStarted(ctx context.Context) error {
	auditLog := domain.NewAuditLog(domain.EventSchedulerStarted, "Message Scheduler Started").
		WithDescription("Message processing scheduler has been started").
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) LogSchedulerStopped(ctx context.Context) error {
	auditLog := domain.NewAuditLog(domain.EventSchedulerStopped, "Message Scheduler Stopped").
		WithDescription("Message processing scheduler has been stopped").
		Build()

	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) Log(ctx context.Context, auditLog *domain.AuditLog) error {
	return s.logWithFallback(ctx, auditLog)
}

func (s *auditService) GetAuditLogs(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	return s.auditRepo.GetAuditLogs(ctx, filter)
}

func (s *auditService) GetBatchAuditLogs(ctx context.Context, batchID string) ([]*domain.AuditLog, error) {
	return s.auditRepo.GetBatchAuditLogs(ctx, batchID)
}

func (s *auditService) GetMessageAuditLogs(ctx context.Context, messageID string) ([]*domain.AuditLog, error) {
	return s.auditRepo.GetMessageAuditLogs(ctx, messageID)
}

func (s *auditService) GetAuditLogStats(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error) {
	return s.auditRepo.GetAuditLogStats(ctx, filter)
}

func (s *auditService) CleanupOldAuditLogs(ctx context.Context, days int) (int64, error) {
	return s.auditRepo.DeleteOldAuditLogs(ctx, days)
}

// logWithFallback attempts to log the audit entry, but falls back to standard logging if it fails
// This ensures that audit logging failures don't break the main application flow
func (s *auditService) logWithFallback(ctx context.Context, auditLog *domain.AuditLog) error {
	err := s.auditRepo.Log(ctx, auditLog)
	if err != nil {
		// Fall back to standard logging if audit logging fails
		description := ""
		if auditLog.Description != nil {
			description = *auditLog.Description
		}
		log.Printf("AUDIT LOG FAILED (fallback to standard log): %s - %s: %s",
			auditLog.EventType, auditLog.EventName, description)
		if auditLog.BatchID != nil {
			log.Printf("  Batch ID: %s", auditLog.BatchID.String())
		}
		if auditLog.MessageID != nil {
			log.Printf("  Message ID: %s", auditLog.MessageID.String())
		}
		return fmt.Errorf("audit logging failed: %w", err)
	}
	return nil
}
