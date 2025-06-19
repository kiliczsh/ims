package domain

import (
	"time"

	"github.com/google/uuid"
)

// MessageStatus represents the status of a message
type MessageStatus string

const (
	StatusPending MessageStatus = "pending"
	StatusSending MessageStatus = "sending"
	StatusSent    MessageStatus = "sent"
	StatusFailed  MessageStatus = "failed"
)

// Message represents a message entity
type Message struct {
	ID          uuid.UUID     `json:"id" db:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	PhoneNumber string        `json:"phone_number" db:"phone_number" example:"+1234567890"`
	Content     string        `json:"content" db:"content" example:"Hello, this is a test message"`
	Status      MessageStatus `json:"status" db:"status" example:"sent" enums:"pending,sending,sent,failed"`
	MessageID   *string       `json:"message_id,omitempty" db:"message_id" example:"msg_12345"`
	RetryCount  int           `json:"retry_count" db:"retry_count" example:"0"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at" example:"2023-12-01T10:00:00Z"`
	SentAt      *time.Time    `json:"sent_at,omitempty" db:"sent_at" example:"2023-12-01T10:05:00Z"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at" example:"2023-12-01T10:05:00Z"`
}

// WebhookRequest represents a request to send a message via webhook
type WebhookRequest struct {
	To      string `json:"to" example:"+1234567890"`
	Content string `json:"content" example:"Hello, this is a test message"`
}

// WebhookResponse represents the response from webhook
type WebhookResponse struct {
	Message   string `json:"message" example:"Message sent successfully"`
	MessageID string `json:"messageId" example:"msg_12345"`
}

// SentMessageResponse represents a successfully sent message in API responses
type SentMessageResponse struct {
	ID          uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	PhoneNumber string    `json:"phone_number" example:"+1234567890"`
	Content     string    `json:"content" example:"Hello, this is a test message"`
	MessageID   string    `json:"message_id" example:"msg_12345"`
	SentAt      time.Time `json:"sent_at" example:"2023-12-01T10:05:00Z"`
}

// SchedulerStatus represents the current status of the scheduler
type SchedulerStatus struct {
	Running   bool       `json:"running" example:"true"`
	StartedAt *time.Time `json:"started_at,omitempty" example:"2023-12-01T10:00:00Z"`
}
