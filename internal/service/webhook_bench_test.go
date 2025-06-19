package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ims/internal/domain"
)

func BenchmarkWebhookClient_Send_Success(b *testing.B) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := domain.WebhookResponse{
			Message:   "Message sent successfully",
			MessageID: "msg-benchmark",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 0)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Send(ctx, "+1234567890", "Benchmark test message")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhookClient_Send_NonJSONResponse(b *testing.B) {
	// Create a test server that returns a non-JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 0)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Send(ctx, "+1234567890", "Benchmark test message")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhookClient_RequestCreation(b *testing.B) {
	client := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 0)
	req := domain.WebhookRequest{
		To:      "+1234567890",
		Content: "Benchmark test message",
	}
	var resp domain.WebhookResponse

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// This will fail due to connection error, but we're benchmarking the request creation part
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		client.doRequest(ctx, req, &resp)
		cancel()
	}
}
