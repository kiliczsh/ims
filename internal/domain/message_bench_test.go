package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testMessageID = "msg_12345"

func BenchmarkMessage_JSONMarshal(b *testing.B) {
	msgID := uuid.New()
	messageID := testMessageID
	sentAt := time.Now()

	message := &Message{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message for JSON marshaling benchmark",
		Status:      StatusPending,
		MessageID:   &messageID,
		RetryCount:  1,
		CreatedAt:   time.Now(),
		SentAt:      &sentAt,
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessage_JSONUnmarshal(b *testing.B) {
	msgID := uuid.New()
	messageID := testMessageID
	sentAt := time.Now()

	original := &Message{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message for JSON unmarshaling benchmark",
		Status:      StatusSent,
		MessageID:   &messageID,
		RetryCount:  1,
		CreatedAt:   time.Now(),
		SentAt:      &sentAt,
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(original)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var message Message
		err := json.Unmarshal(data, &message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhookRequest_JSONMarshal(b *testing.B) {
	req := &WebhookRequest{
		To:      "+1234567890",
		Content: "Hello, this is a benchmark test message",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuditLog_Creation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		batchID := uuid.New()
		messageID := uuid.New()
		requestID := uuid.New().String()

		log := NewAuditLog(EventMessageSent, "Message Sent Successfully").
			WithDescription("Benchmark test audit log creation").
			WithBatchID(batchID).
			WithMessageID(messageID).
			WithRequestID(requestID).
			WithDuration(100*time.Millisecond).
			WithHTTPDetails("POST", "/api/webhook", 200).
			WithMessageCounts(10, 8, 2).
			WithMetadata("benchmark", true).
			Build()

		_ = log
	}
}

func BenchmarkAuditLog_JSONMarshal(b *testing.B) {
	batchID := uuid.New()
	messageID := uuid.New()
	requestID := uuid.New().String()

	log := NewAuditLog(EventMessageSent, "Message Sent Successfully").
		WithDescription("Benchmark test audit log creation").
		WithBatchID(batchID).
		WithMessageID(messageID).
		WithRequestID(requestID).
		WithDuration(100*time.Millisecond).
		WithHTTPDetails("POST", "/api/webhook", 200).
		WithMessageCounts(10, 8, 2).
		WithMetadata("benchmark", true).
		Build()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(log)
		if err != nil {
			b.Fatal(err)
		}
	}
}
