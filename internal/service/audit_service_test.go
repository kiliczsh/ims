package service

import (
	"context"
	"errors"
	"ims/internal/domain"
	"ims/internal/repository"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAuditService(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	// Test that the service implements the interface
	var _ AuditService = service
}

func TestAuditService_LogBatchStarted(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	batchID := uuid.New()
	messageCount := 5

	ctx := context.Background()
	err := service.LogBatchStarted(ctx, batchID, messageCount)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if auditRepo.Count() != 1 {
		t.Errorf("Expected 1 audit log, got %d", auditRepo.Count())
	}

	// Get the logged entry
	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.EventType != domain.EventBatchStarted {
		t.Errorf("Expected event type %s, got %s", domain.EventBatchStarted, log.EventType)
	}

	if log.EventName != "Batch Processing Started" {
		t.Errorf("Expected event name 'Batch Processing Started', got %s", log.EventName)
	}

	if *log.BatchID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, *log.BatchID)
	}

	if *log.MessageCount != messageCount {
		t.Errorf("Expected message count %d, got %d", messageCount, *log.MessageCount)
	}
}

func TestAuditService_LogBatchCompleted(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	batchID := uuid.New()
	duration := 5 * time.Second
	successCount := 8
	failureCount := 2

	ctx := context.Background()
	err := service.LogBatchCompleted(ctx, batchID, duration, successCount, failureCount)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.EventType != domain.EventBatchCompleted {
		t.Errorf("Expected event type %s, got %s", domain.EventBatchCompleted, log.EventType)
	}

	if *log.BatchID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, *log.BatchID)
	}

	if *log.DurationMs != 5000 {
		t.Errorf("Expected duration 5000ms, got %d", *log.DurationMs)
	}

	if *log.MessageCount != 10 {
		t.Errorf("Expected total message count 10, got %d", *log.MessageCount)
	}

	if *log.SuccessCount != successCount {
		t.Errorf("Expected success count %d, got %d", successCount, *log.SuccessCount)
	}

	if *log.FailureCount != failureCount {
		t.Errorf("Expected failure count %d, got %d", failureCount, *log.FailureCount)
	}
}

func TestAuditService_LogBatchFailed(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	batchID := uuid.New()
	duration := 2 * time.Second
	testError := errors.New("batch processing failed")

	ctx := context.Background()
	err := service.LogBatchFailed(ctx, batchID, duration, testError)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventBatchFailed {
		t.Errorf("Expected event type %s, got %s", domain.EventBatchFailed, log.EventType)
	}

	if *log.BatchID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, *log.BatchID)
	}

	if *log.DurationMs != 2000 {
		t.Errorf("Expected duration 2000ms, got %d", *log.DurationMs)
	}

	// Check metadata contains error
	if errorMsg, exists := log.Metadata["error"]; !exists || errorMsg != testError.Error() {
		t.Errorf("Expected error in metadata to be '%s', got %v", testError.Error(), errorMsg)
	}
}

func TestAuditService_LogMessageSent(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	messageID := uuid.New()
	duration := 100 * time.Millisecond
	webhookURL := "https://example.com/webhook"

	ctx := context.Background()
	err := service.LogMessageSent(ctx, messageID, duration, webhookURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventMessageSent {
		t.Errorf("Expected event type %s, got %s", domain.EventMessageSent, log.EventType)
	}

	if *log.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *log.MessageID)
	}

	if *log.DurationMs != 100 {
		t.Errorf("Expected duration 100ms, got %d", *log.DurationMs)
	}

	if url, exists := log.Metadata["webhook_url"]; !exists || url != webhookURL {
		t.Errorf("Expected webhook_url in metadata to be '%s', got %v", webhookURL, url)
	}
}

func TestAuditService_LogMessageFailed(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	messageID := uuid.New()
	duration := 50 * time.Millisecond
	webhookURL := "https://example.com/webhook"
	testError := errors.New("webhook timeout")

	ctx := context.Background()
	err := service.LogMessageFailed(ctx, messageID, duration, webhookURL, testError)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventMessageFailed {
		t.Errorf("Expected event type %s, got %s", domain.EventMessageFailed, log.EventType)
	}

	if *log.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *log.MessageID)
	}

	if url, exists := log.Metadata["webhook_url"]; !exists || url != webhookURL {
		t.Errorf("Expected webhook_url in metadata to be '%s', got %v", webhookURL, url)
	}

	if errorMsg, exists := log.Metadata["error"]; !exists || errorMsg != testError.Error() {
		t.Errorf("Expected error in metadata to be '%s', got %v", testError.Error(), errorMsg)
	}
}

func TestAuditService_LogWebhookRequest(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	messageID := uuid.New()
	webhookURL := "https://example.com/webhook"
	method := "POST"
	requestBody := map[string]interface{}{
		"to":      "+1234567890",
		"content": "Test message",
	}

	ctx := context.Background()
	err := service.LogWebhookRequest(ctx, messageID, webhookURL, method, requestBody)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventWebhookRequest {
		t.Errorf("Expected event type %s, got %s", domain.EventWebhookRequest, log.EventType)
	}

	if *log.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *log.MessageID)
	}

	if *log.HTTPMethod != method {
		t.Errorf("Expected HTTP method %s, got %s", method, *log.HTTPMethod)
	}

	if *log.Endpoint != webhookURL {
		t.Errorf("Expected endpoint %s, got %s", webhookURL, *log.Endpoint)
	}

	if body, exists := log.Metadata["request_body"]; !exists {
		t.Error("Expected request_body in metadata")
	} else {
		bodyMap, ok := body.(map[string]interface{})
		if !ok {
			t.Error("Expected request_body to be a map")
		} else if bodyMap["to"] != "+1234567890" {
			t.Errorf("Expected request_body.to to be '+1234567890', got %v", bodyMap["to"])
		}
	}
}

func TestAuditService_LogWebhookResponse(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	messageID := uuid.New()
	webhookURL := "https://example.com/webhook"
	statusCode := 200
	duration := 150 * time.Millisecond
	responseBody := map[string]interface{}{
		"message":   "Message sent successfully",
		"messageId": "msg_123",
	}

	ctx := context.Background()
	err := service.LogWebhookResponse(ctx, messageID, webhookURL, statusCode, duration, responseBody)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventWebhookResponse {
		t.Errorf("Expected event type %s, got %s", domain.EventWebhookResponse, log.EventType)
	}

	if *log.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *log.MessageID)
	}

	if *log.StatusCode != statusCode {
		t.Errorf("Expected status code %d, got %d", statusCode, *log.StatusCode)
	}

	if *log.DurationMs != 150 {
		t.Errorf("Expected duration 150ms, got %d", *log.DurationMs)
	}
}

func TestAuditService_LogAPIRequest(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	requestID := "req_123"
	method := "GET"
	endpoint := "/api/messages"
	statusCode := 200
	duration := 25 * time.Millisecond
	userAgent := "Test-Agent/1.0"

	ctx := context.Background()
	err := service.LogAPIRequest(ctx, requestID, method, endpoint, statusCode, duration, userAgent)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventAPIRequest {
		t.Errorf("Expected event type %s, got %s", domain.EventAPIRequest, log.EventType)
	}

	if *log.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, *log.RequestID)
	}

	if *log.HTTPMethod != method {
		t.Errorf("Expected HTTP method %s, got %s", method, *log.HTTPMethod)
	}

	if *log.Endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, *log.Endpoint)
	}

	if *log.StatusCode != statusCode {
		t.Errorf("Expected status code %d, got %d", statusCode, *log.StatusCode)
	}

	if agent, exists := log.Metadata["user_agent"]; !exists || agent != userAgent {
		t.Errorf("Expected user_agent in metadata to be '%s', got %v", userAgent, agent)
	}
}

func TestAuditService_LogSchedulerStarted(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	ctx := context.Background()
	err := service.LogSchedulerStarted(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventSchedulerStarted {
		t.Errorf("Expected event type %s, got %s", domain.EventSchedulerStarted, log.EventType)
	}

	if log.EventName != "Message Scheduler Started" {
		t.Errorf("Expected event name 'Message Scheduler Started', got %s", log.EventName)
	}
}

func TestAuditService_LogSchedulerStopped(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	ctx := context.Background()
	err := service.LogSchedulerStopped(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	log := logs[0]
	if log.EventType != domain.EventSchedulerStopped {
		t.Errorf("Expected event type %s, got %s", domain.EventSchedulerStopped, log.EventType)
	}
}

func TestAuditService_Log_Generic(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	customLog := domain.NewAuditLog(domain.EventAPIRequest, "Custom Event").
		WithDescription("Custom audit log entry").
		Build()

	ctx := context.Background()
	err := service.Log(ctx, customLog)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logs, err := auditRepo.GetAuditLogs(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.EventName != "Custom Event" {
		t.Errorf("Expected event name 'Custom Event', got %s", log.EventName)
	}
}

func TestAuditService_GetAuditLogs(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	// Add some test logs
	log1 := domain.NewAuditLog(domain.EventMessageSent, "Message 1").Build()
	log2 := domain.NewAuditLog(domain.EventMessageFailed, "Message 2").Build()
	auditRepo.AddLog(log1)
	auditRepo.AddLog(log2)

	ctx := context.Background()
	filter := &domain.AuditLogFilter{
		EventTypes: []domain.AuditEventType{domain.EventMessageSent},
	}

	logs, err := service.GetAuditLogs(ctx, filter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 filtered log, got %d", len(logs))
	}

	if logs[0].EventType != domain.EventMessageSent {
		t.Errorf("Expected EventMessageSent, got %s", logs[0].EventType)
	}
}

func TestAuditService_GetBatchAuditLogs(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	batchID := uuid.New()
	log1 := domain.NewAuditLog(domain.EventBatchStarted, "Batch Started").
		WithBatchID(batchID).Build()
	log2 := domain.NewAuditLog(domain.EventBatchCompleted, "Batch Completed").
		WithBatchID(batchID).Build()
	auditRepo.AddLog(log1)
	auditRepo.AddLog(log2)

	ctx := context.Background()
	logs, err := service.GetBatchAuditLogs(ctx, batchID.String())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 batch logs, got %d", len(logs))
	}
}

func TestAuditService_GetMessageAuditLogs(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	messageID := uuid.New()
	log1 := domain.NewAuditLog(domain.EventMessageSent, "Message Sent").
		WithMessageID(messageID).Build()
	log2 := domain.NewAuditLog(domain.EventWebhookResponse, "Webhook Response").
		WithMessageID(messageID).Build()
	auditRepo.AddLog(log1)
	auditRepo.AddLog(log2)

	ctx := context.Background()
	logs, err := service.GetMessageAuditLogs(ctx, messageID.String())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 message logs, got %d", len(logs))
	}
}

func TestAuditService_GetAuditLogStats(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	// Add some test logs
	log1 := domain.NewAuditLog(domain.EventMessageSent, "Message 1").Build()
	log2 := domain.NewAuditLog(domain.EventMessageSent, "Message 2").Build()
	log3 := domain.NewAuditLog(domain.EventMessageFailed, "Message 3").Build()
	auditRepo.AddLog(log1)
	auditRepo.AddLog(log2)
	auditRepo.AddLog(log3)

	ctx := context.Background()
	stats, err := service.GetAuditLogStats(ctx, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", stats.TotalCount)
	}

	if stats.EventTypeCounts[domain.EventMessageSent] != 2 {
		t.Errorf("Expected 2 EventMessageSent, got %d", stats.EventTypeCounts[domain.EventMessageSent])
	}

	if stats.EventTypeCounts[domain.EventMessageFailed] != 1 {
		t.Errorf("Expected 1 EventMessageFailed, got %d", stats.EventTypeCounts[domain.EventMessageFailed])
	}
}

func TestAuditService_CleanupOldAuditLogs(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	// Add some test logs
	log1 := domain.NewAuditLog(domain.EventMessageSent, "Message 1").Build()
	auditRepo.AddLog(log1)

	ctx := context.Background()
	deleted, err := service.CleanupOldAuditLogs(ctx, 30)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should not delete recent logs (mock implementation doesn't simulate old dates)
	if deleted != 0 {
		t.Errorf("Expected 0 deleted logs, got %d", deleted)
	}
}

func TestAuditService_RepositoryError(t *testing.T) {
	auditRepo := repository.NewMockAuditRepository()
	service := NewAuditService(auditRepo)

	// Configure repository to return error
	expectedError := errors.New("database error")
	auditRepo.LogFunc = func(ctx context.Context, auditLog *domain.AuditLog) error {
		return expectedError
	}

	ctx := context.Background()
	err := service.LogSchedulerStarted(ctx)

	// Should not return error due to fallback logging
	if err != nil {
		t.Errorf("Expected no error due to fallback, got %v", err)
	}
}
