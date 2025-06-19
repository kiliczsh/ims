package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuditEventType_Constants(t *testing.T) {
	tests := []struct {
		name      string
		eventType AuditEventType
		expected  string
	}{
		{"EventBatchStarted", EventBatchStarted, "batch_started"},
		{"EventBatchCompleted", EventBatchCompleted, "batch_completed"},
		{"EventBatchFailed", EventBatchFailed, "batch_failed"},
		{"EventMessageSent", EventMessageSent, "message_sent"},
		{"EventMessageFailed", EventMessageFailed, "message_failed"},
		{"EventSchedulerStarted", EventSchedulerStarted, "scheduler_started"},
		{"EventSchedulerStopped", EventSchedulerStopped, "scheduler_stopped"},
		{"EventAPIRequest", EventAPIRequest, "api_request"},
		{"EventWebhookRequest", EventWebhookRequest, "webhook_request"},
		{"EventWebhookResponse", EventWebhookResponse, "webhook_response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
			}
		})
	}
}

func TestNewAuditLog(t *testing.T) {
	eventType := EventBatchStarted
	eventName := "Test Event"

	builder := NewAuditLog(eventType, eventName)
	log := builder.Build()

	if log.EventType != eventType {
		t.Errorf("Expected event type %s, got %s", eventType, log.EventType)
	}

	if log.EventName != eventName {
		t.Errorf("Expected event name %s, got %s", eventName, log.EventName)
	}

	if log.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}

	if log.CreatedAt.IsZero() {
		t.Error("Expected non-zero created at time")
	}

	if log.Metadata == nil {
		t.Error("Expected metadata map to be initialized")
	}
}

func TestAuditLogBuilder(t *testing.T) {
	batchID := uuid.New()
	messageID := uuid.New()
	requestID := "req-123"
	description := "Test description"
	duration := 100 * time.Millisecond

	builder := NewAuditLog(EventBatchStarted, "Test Event").
		WithDescription(description).
		WithBatchID(batchID).
		WithMessageID(messageID).
		WithRequestID(requestID).
		WithHTTPDetails("POST", "/api/test", 200).
		WithDuration(duration).
		WithMessageCounts(10, 8, 2).
		WithMetadata("key1", "value1").
		WithMetadataMap(map[string]interface{}{"key2": "value2", "key3": 42})

	log := builder.Build()

	// Test all fields
	if *log.Description != description {
		t.Errorf("Expected description %s, got %s", description, *log.Description)
	}

	if *log.BatchID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, *log.BatchID)
	}

	if *log.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *log.MessageID)
	}

	if *log.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, *log.RequestID)
	}

	if *log.HTTPMethod != "POST" {
		t.Errorf("Expected HTTP method POST, got %s", *log.HTTPMethod)
	}

	if *log.Endpoint != "/api/test" {
		t.Errorf("Expected endpoint /api/test, got %s", *log.Endpoint)
	}

	if *log.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", *log.StatusCode)
	}

	if *log.DurationMs != 100 {
		t.Errorf("Expected duration 100ms, got %d", *log.DurationMs)
	}

	if *log.MessageCount != 10 {
		t.Errorf("Expected message count 10, got %d", *log.MessageCount)
	}

	if *log.SuccessCount != 8 {
		t.Errorf("Expected success count 8, got %d", *log.SuccessCount)
	}

	if *log.FailureCount != 2 {
		t.Errorf("Expected failure count 2, got %d", *log.FailureCount)
	}

	// Test metadata
	if log.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1 to be 'value1', got %v", log.Metadata["key1"])
	}

	if log.Metadata["key2"] != "value2" {
		t.Errorf("Expected metadata key2 to be 'value2', got %v", log.Metadata["key2"])
	}

	if log.Metadata["key3"] != 42 {
		t.Errorf("Expected metadata key3 to be 42, got %v", log.Metadata["key3"])
	}
}

func TestAuditLog_MarshalJSON(t *testing.T) {
	log := NewAuditLog(EventMessageSent, "Message Sent").
		WithDescription("Test message").
		WithMetadata("webhook_url", "https://example.com").
		WithMetadata("retry_count", 1).
		Build()

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal audit log: %v", err)
	}

	// Unmarshal back to verify
	var unmarshaled AuditLog
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal audit log: %v", err)
	}

	if unmarshaled.EventType != log.EventType {
		t.Errorf("Expected event type %s, got %s", log.EventType, unmarshaled.EventType)
	}

	if unmarshaled.EventName != log.EventName {
		t.Errorf("Expected event name %s, got %s", log.EventName, unmarshaled.EventName)
	}
}

func TestAuditLog_MarshalJSON_EmptyMetadata(t *testing.T) {
	log := NewAuditLog(EventSchedulerStarted, "Scheduler Started").Build()

	// Clear metadata to test empty case
	log.Metadata = nil

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal audit log with empty metadata: %v", err)
	}

	// Should not contain metadata field when empty
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := jsonMap["metadata"]; exists {
		t.Error("Expected metadata field to be omitted when empty")
	}
}

func TestAuditLogFilter(t *testing.T) {
	batchID := uuid.New()
	messageID := uuid.New()
	requestID := "req-123"
	endpoint := "/api/messages"
	now := time.Now()

	filter := &AuditLogFilter{
		EventTypes: []AuditEventType{EventMessageSent, EventMessageFailed},
		BatchID:    &batchID,
		MessageID:  &messageID,
		RequestID:  &requestID,
		Endpoint:   &endpoint,
		FromDate:   &now,
		ToDate:     &now,
		Limit:      100,
		Offset:     0,
	}

	// Test that all fields are set correctly
	if len(filter.EventTypes) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(filter.EventTypes))
	}

	if *filter.BatchID != batchID {
		t.Errorf("Expected batch ID %s, got %s", batchID, *filter.BatchID)
	}

	if *filter.MessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, *filter.MessageID)
	}

	if *filter.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, *filter.RequestID)
	}

	if *filter.Endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, *filter.Endpoint)
	}

	if filter.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", filter.Limit)
	}

	if filter.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", filter.Offset)
	}
}

func TestAuditLogStats(t *testing.T) {
	lastEventTime := "2023-12-01T10:00:00Z"
	avgDuration := 150.5

	stats := &AuditLogStats{
		TotalCount: 100,
		EventTypeCounts: map[AuditEventType]int64{
			EventMessageSent:   50,
			EventMessageFailed: 10,
			EventBatchStarted:  20,
		},
		LastEventTime:          &lastEventTime,
		AverageRequestDuration: &avgDuration,
	}

	if stats.TotalCount != 100 {
		t.Errorf("Expected total count 100, got %d", stats.TotalCount)
	}

	if stats.EventTypeCounts[EventMessageSent] != 50 {
		t.Errorf("Expected message sent count 50, got %d", stats.EventTypeCounts[EventMessageSent])
	}

	if *stats.LastEventTime != lastEventTime {
		t.Errorf("Expected last event time %s, got %s", lastEventTime, *stats.LastEventTime)
	}

	if *stats.AverageRequestDuration != avgDuration {
		t.Errorf("Expected average duration %f, got %f", avgDuration, *stats.AverageRequestDuration)
	}
}
