package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func BenchmarkMessage_JSONMarshal(b *testing.B) {
	msgID := uuid.New()
	messageID := "msg_12345"
	sentAt := time.Now()

	msg := &Message{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message for benchmarking",
		Status:      StatusSent,
		MessageID:   &messageID,
		RetryCount:  1,
		CreatedAt:   time.Now(),
		SentAt:      &sentAt,
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessage_JSONUnmarshal(b *testing.B) {
	msgID := uuid.New()
	messageID := "msg_12345"
	sentAt := time.Now()

	msg := &Message{
		ID:          msgID,
		PhoneNumber: "+1234567890",
		Content:     "Test message for benchmarking",
		Status:      StatusSent,
		MessageID:   &messageID,
		RetryCount:  1,
		CreatedAt:   time.Now(),
		SentAt:      &sentAt,
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var unmarshaled Message
		err := json.Unmarshal(data, &unmarshaled)
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
