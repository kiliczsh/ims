// Package domain contains the core business entities and domain logic for the IMS application.
// It defines audit logs, messages, errors, and other business domain types.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditEventType string

const (
	EventBatchStarted     AuditEventType = "batch_started"
	EventBatchCompleted   AuditEventType = "batch_completed"
	EventBatchFailed      AuditEventType = "batch_failed"
	EventMessageSent      AuditEventType = "message_sent"
	EventMessageFailed    AuditEventType = "message_failed"
	EventSchedulerStarted AuditEventType = "scheduler_started"
	EventSchedulerStopped AuditEventType = "scheduler_stopped"
	EventAPIRequest       AuditEventType = "api_request"
	EventWebhookRequest   AuditEventType = "webhook_request"
	EventWebhookResponse  AuditEventType = "webhook_response"
)

type AuditLog struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	EventType   AuditEventType `json:"event_type" db:"event_type"`
	EventName   string         `json:"event_name" db:"event_name"`
	Description *string        `json:"description,omitempty" db:"description"`

	// Context information
	BatchID   *uuid.UUID `json:"batch_id,omitempty" db:"batch_id"`
	MessageID *uuid.UUID `json:"message_id,omitempty" db:"message_id"`
	RequestID *string    `json:"request_id,omitempty" db:"request_id"`

	// Request/Response details
	HTTPMethod *string `json:"http_method,omitempty" db:"http_method"`
	Endpoint   *string `json:"endpoint,omitempty" db:"endpoint"`
	StatusCode *int    `json:"status_code,omitempty" db:"status_code"`

	// Metrics
	DurationMs   *int `json:"duration_ms,omitempty" db:"duration_ms"`
	MessageCount *int `json:"message_count,omitempty" db:"message_count"`
	SuccessCount *int `json:"success_count,omitempty" db:"success_count"`
	FailureCount *int `json:"failure_count,omitempty" db:"failure_count"`

	// Additional data (JSON)
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// AuditLogStats represents statistics about audit logs
type AuditLogStats struct {
	TotalCount             int64                    `json:"total_count"`
	EventTypeCounts        map[AuditEventType]int64 `json:"event_type_counts"`
	LastEventTime          *string                  `json:"last_event_time,omitempty"`
	AverageRequestDuration *float64                 `json:"average_request_duration,omitempty"`
}

// AuditLogBuilder helps build audit log entries
type AuditLogBuilder struct {
	log *AuditLog
}

func NewAuditLog(eventType AuditEventType, eventName string) *AuditLogBuilder {
	return &AuditLogBuilder{
		log: &AuditLog{
			ID:        uuid.New(),
			EventType: eventType,
			EventName: eventName,
			CreatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		},
	}
}

func (b *AuditLogBuilder) WithDescription(desc string) *AuditLogBuilder {
	b.log.Description = &desc
	return b
}

func (b *AuditLogBuilder) WithBatchID(batchID uuid.UUID) *AuditLogBuilder {
	b.log.BatchID = &batchID
	return b
}

func (b *AuditLogBuilder) WithMessageID(messageID uuid.UUID) *AuditLogBuilder {
	b.log.MessageID = &messageID
	return b
}

func (b *AuditLogBuilder) WithRequestID(requestID string) *AuditLogBuilder {
	b.log.RequestID = &requestID
	return b
}

func (b *AuditLogBuilder) WithHTTPDetails(method, endpoint string, statusCode int) *AuditLogBuilder {
	b.log.HTTPMethod = &method
	b.log.Endpoint = &endpoint
	b.log.StatusCode = &statusCode
	return b
}

func (b *AuditLogBuilder) WithDuration(duration time.Duration) *AuditLogBuilder {
	durationMs := int(duration.Milliseconds())
	b.log.DurationMs = &durationMs
	return b
}

func (b *AuditLogBuilder) WithMessageCounts(total, success, failure int) *AuditLogBuilder {
	b.log.MessageCount = &total
	b.log.SuccessCount = &success
	b.log.FailureCount = &failure
	return b
}

func (b *AuditLogBuilder) WithMetadata(key string, value interface{}) *AuditLogBuilder {
	b.log.Metadata[key] = value
	return b
}

func (b *AuditLogBuilder) WithMetadataMap(metadata map[string]interface{}) *AuditLogBuilder {
	for k, v := range metadata {
		b.log.Metadata[k] = v
	}
	return b
}

func (b *AuditLogBuilder) Build() *AuditLog {
	return b.log
}

// AuditLogFilter for querying audit logs
type AuditLogFilter struct {
	EventTypes []AuditEventType `json:"event_types,omitempty"`
	BatchID    *uuid.UUID       `json:"batch_id,omitempty"`
	MessageID  *uuid.UUID       `json:"message_id,omitempty"`
	RequestID  *string          `json:"request_id,omitempty"`
	Endpoint   *string          `json:"endpoint,omitempty"`
	FromDate   *time.Time       `json:"from_date,omitempty"`
	ToDate     *time.Time       `json:"to_date,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	Offset     int              `json:"offset,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for the AuditLog metadata field
func (a *AuditLog) MarshalJSON() ([]byte, error) {
	type Alias AuditLog
	aux := &struct {
		Metadata json.RawMessage `json:"metadata,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if len(a.Metadata) > 0 {
		metadataBytes, err := json.Marshal(a.Metadata)
		if err != nil {
			return nil, err
		}
		aux.Metadata = metadataBytes
	}

	return json.Marshal(aux)
}
