package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMessageStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   MessageStatus
		expected string
	}{
		{"StatusPending", StatusPending, "pending"},
		{"StatusSending", StatusSending, "sending"},
		{"StatusSent", StatusSent, "sent"},
		{"StatusFailed", StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestMessage_JSON(t *testing.T) {
	msgID := uuid.New()
	messageID := "msg_12345"
	sentAt := time.Now()

	msg := &Message{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      StatusSent,
		MessageID:   &messageID,
		RetryCount:  1,
		CreatedAt:   time.Now(),
		SentAt:      &sentAt,
		UpdatedAt:   time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify all fields
	if unmarshaled.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, unmarshaled.ID)
	}

	if unmarshaled.PhoneNumber != msg.PhoneNumber {
		t.Errorf("Expected phone number %s, got %s", msg.PhoneNumber, unmarshaled.PhoneNumber)
	}

	if unmarshaled.Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, unmarshaled.Content)
	}

	if unmarshaled.Status != msg.Status {
		t.Errorf("Expected status %s, got %s", msg.Status, unmarshaled.Status)
	}

	if *unmarshaled.MessageID != *msg.MessageID {
		t.Errorf("Expected message ID %s, got %s", *msg.MessageID, *unmarshaled.MessageID)
	}

	if unmarshaled.RetryCount != msg.RetryCount {
		t.Errorf("Expected retry count %d, got %d", msg.RetryCount, unmarshaled.RetryCount)
	}
}

func TestMessage_WithNilValues(t *testing.T) {
	msg := &Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      StatusPending,
		MessageID:   nil,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		SentAt:      nil,
		UpdatedAt:   time.Now(),
	}

	// Test JSON marshaling with nil values
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message with nil values: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal message with nil values: %v", err)
	}

	if unmarshaled.MessageID != nil {
		t.Errorf("Expected nil message ID, got %v", unmarshaled.MessageID)
	}

	if unmarshaled.SentAt != nil {
		t.Errorf("Expected nil sent at, got %v", unmarshaled.SentAt)
	}
}

func TestWebhookRequest(t *testing.T) {
	req := &WebhookRequest{
		To:      "+1234567890",
		Content: "Hello, this is a test message",
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal webhook request: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled WebhookRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal webhook request: %v", err)
	}

	if unmarshaled.To != req.To {
		t.Errorf("Expected To %s, got %s", req.To, unmarshaled.To)
	}

	if unmarshaled.Content != req.Content {
		t.Errorf("Expected Content %s, got %s", req.Content, unmarshaled.Content)
	}
}

func TestWebhookResponse(t *testing.T) {
	resp := &WebhookResponse{
		Message:   "Message sent successfully",
		MessageID: "msg_12345",
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal webhook response: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled WebhookResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal webhook response: %v", err)
	}

	if unmarshaled.Message != resp.Message {
		t.Errorf("Expected Message %s, got %s", resp.Message, unmarshaled.Message)
	}

	if unmarshaled.MessageID != resp.MessageID {
		t.Errorf("Expected MessageID %s, got %s", resp.MessageID, unmarshaled.MessageID)
	}
}

func TestSentMessageResponse(t *testing.T) {
	msgID := uuid.New()
	sentAt := time.Now()

	resp := &SentMessageResponse{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		MessageID:   "msg_12345",
		SentAt:      sentAt,
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal sent message response: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SentMessageResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal sent message response: %v", err)
	}

	if unmarshaled.ID != resp.ID {
		t.Errorf("Expected ID %s, got %s", resp.ID, unmarshaled.ID)
	}

	if unmarshaled.PhoneNumber != resp.PhoneNumber {
		t.Errorf("Expected PhoneNumber %s, got %s", resp.PhoneNumber, unmarshaled.PhoneNumber)
	}

	if unmarshaled.Content != resp.Content {
		t.Errorf("Expected Content %s, got %s", resp.Content, unmarshaled.Content)
	}

	if unmarshaled.MessageID != resp.MessageID {
		t.Errorf("Expected MessageID %s, got %s", resp.MessageID, unmarshaled.MessageID)
	}

	// Allow for small time differences in JSON round-trip
	timeDiff := unmarshaled.SentAt.Sub(resp.SentAt)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("SentAt time difference too large: %v", timeDiff)
	}
}

func TestSchedulerStatus(t *testing.T) {
	startedAt := time.Now()

	tests := []struct {
		name   string
		status *SchedulerStatus
	}{
		{
			name: "Running scheduler",
			status: &SchedulerStatus{
				Running:   true,
				StartedAt: &startedAt,
			},
		},
		{
			name: "Stopped scheduler",
			status: &SchedulerStatus{
				Running:   false,
				StartedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Failed to marshal scheduler status: %v", err)
			}

			// Test JSON unmarshaling
			var unmarshaled SchedulerStatus
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal scheduler status: %v", err)
			}

			if unmarshaled.Running != tt.status.Running {
				t.Errorf("Expected Running %t, got %t", tt.status.Running, unmarshaled.Running)
			}

			if tt.status.StartedAt == nil {
				if unmarshaled.StartedAt != nil {
					t.Errorf("Expected nil StartedAt, got %v", unmarshaled.StartedAt)
				}
			} else {
				if unmarshaled.StartedAt == nil {
					t.Error("Expected non-nil StartedAt")
				} else {
					// Allow for small time differences in JSON round-trip
					timeDiff := unmarshaled.StartedAt.Sub(*tt.status.StartedAt)
					if timeDiff > time.Second || timeDiff < -time.Second {
						t.Errorf("StartedAt time difference too large: %v", timeDiff)
					}
				}
			}
		})
	}
}
