package service

import (
	"context"
	"encoding/json"
	"errors"
	"ims/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewWebhookClient(t *testing.T) {
	url := "https://example.com/webhook"
	authKey := "test-auth-key"
	timeout := 30 * time.Second
	maxRetries := 3

	client := NewWebhookClient(url, authKey, timeout, maxRetries)

	if client.url != url {
		t.Errorf("Expected URL %s, got %s", url, client.url)
	}

	if client.authKey != authKey {
		t.Errorf("Expected auth key %s, got %s", authKey, client.authKey)
	}

	if client.maxRetries != maxRetries {
		t.Errorf("Expected max retries %d, got %d", maxRetries, client.maxRetries)
	}

	if client.client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.client.Timeout)
	}
}

func TestWebhookClient_Send_Success(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("x-ins-auth-key") != "test-key" {
			t.Errorf("Expected auth key test-key, got %s", r.Header.Get("x-ins-auth-key"))
		}

		// Verify request body
		var req domain.WebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.To != "+1234567890" {
			t.Errorf("Expected To +1234567890, got %s", req.To)
		}

		if req.Content != "Test message" {
			t.Errorf("Expected Content 'Test message', got %s", req.Content)
		}

		// Return successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := domain.WebhookResponse{
			Message:   "Message sent successfully",
			MessageID: "msg-123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 3)

	ctx := context.Background()
	resp, err := client.Send(ctx, "+1234567890", "Test message")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Message != "Message sent successfully" {
		t.Errorf("Expected message 'Message sent successfully', got %s", resp.Message)
	}

	if resp.MessageID != "msg-123" {
		t.Errorf("Expected message ID 'msg-123', got %s", resp.MessageID)
	}
}

func TestWebhookClient_Send_NonJSONResponse(t *testing.T) {
	// Create a test server that returns a non-JSON response (like webhook.site)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 3)

	ctx := context.Background()
	resp, err := client.Send(ctx, "+1234567890", "Test message")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create a mock response when JSON decoding fails
	if resp.Message != "Accepted" {
		t.Errorf("Expected message 'Accepted', got %s", resp.Message)
	}

	if !strings.HasPrefix(resp.MessageID, "webhook-") {
		t.Errorf("Expected message ID to start with 'webhook-', got %s", resp.MessageID)
	}
}

func TestWebhookClient_Send_HTTPError(t *testing.T) {
	// Create a test server that returns an error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 0) // No retries

	ctx := context.Background()
	_, err := client.Send(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	expectedError := "unexpected status code: 500"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %v", expectedError, err)
	}
}

func TestWebhookClient_Send_Retry(t *testing.T) {
	attempts := 0

	// Create a test server that fails first few times, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success on third attempt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := domain.WebhookResponse{
			Message:   "Message sent successfully",
			MessageID: "msg-123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 3)

	ctx := context.Background()
	start := time.Now()
	resp, err := client.Send(ctx, "+1234567890", "Test message")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error after retries, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if resp.MessageID != "msg-123" {
		t.Errorf("Expected message ID 'msg-123', got %s", resp.MessageID)
	}

	// Should have taken at least 3 seconds due to backoff (1s + 2s)
	if duration < 3*time.Second {
		t.Errorf("Expected at least 3 seconds due to backoff, got %v", duration)
	}
}

func TestWebhookClient_Send_MaxRetriesExceeded(t *testing.T) {
	attempts := 0

	// Create a test server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 2) // Max 2 retries

	ctx := context.Background()
	_, err := client.Send(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error after max retries, got nil")
	}

	// Should attempt 3 times (initial + 2 retries)
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	expectedError := "failed after 3 attempts"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %v", expectedError, err)
	}
}

func TestWebhookClient_Send_ContextCanceled(t *testing.T) {
	// Create a test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 2)

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := client.Send(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error due to context cancellation, got nil")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestWebhookClient_Send_ContextTimeout(t *testing.T) {
	// Create a test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 0) // No retries

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Send(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error due to context timeout, got nil")
	}

	// Check if the error is wrapped context.DeadlineExceeded
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestWebhookClient_Send_InvalidJSON(t *testing.T) {
	// This test is mainly for coverage, as JSON marshaling of WebhookRequest should always succeed
	client := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 0)

	ctx := context.Background()

	// Test with normal values (should succeed)
	var resp domain.WebhookResponse
	err := client.doRequest(ctx, domain.WebhookRequest{
		To:      "+1234567890",
		Content: "Test message",
	}, &resp)

	// This will fail due to connection error, but not JSON marshaling error
	if err == nil {
		t.Fatal("Expected connection error, got nil")
	}

	// Verify it's a connection error, not a JSON error
	if strings.Contains(err.Error(), "marshal") {
		t.Errorf("Unexpected JSON marshal error: %v", err)
	}
}

func TestWebhookClient_Send_AcceptedStatus(t *testing.T) {
	// Create a test server that returns 202 Accepted
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		resp := domain.WebhookResponse{
			Message:   "Message accepted",
			MessageID: "msg-accepted",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 0)

	ctx := context.Background()
	resp, err := client.Send(ctx, "+1234567890", "Test message")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.MessageID != "msg-accepted" {
		t.Errorf("Expected message ID 'msg-accepted', got %s", resp.MessageID)
	}
}

func TestWebhookClient_BackoffLogic(t *testing.T) {
	attempts := 0
	var attemptTimes []time.Time

	// Create a test server that records attempt times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())
		w.WriteHeader(http.StatusInternalServerError) // Always fail
	}))
	defer server.Close()

	client := NewWebhookClient(server.URL, "test-key", 30*time.Second, 2)

	ctx := context.Background()
	start := time.Now()
	_, err := client.Send(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if len(attemptTimes) != 3 {
		t.Fatalf("Expected 3 attempts, got %d", len(attemptTimes))
	}

	// Check backoff timing
	// First attempt: immediate
	// Second attempt: ~1 second later
	// Third attempt: ~2 seconds after second

	firstToSecond := attemptTimes[1].Sub(attemptTimes[0])
	if firstToSecond < 900*time.Millisecond || firstToSecond > 1100*time.Millisecond {
		t.Errorf("Expected ~1s backoff between first and second attempt, got %v", firstToSecond)
	}

	secondToThird := attemptTimes[2].Sub(attemptTimes[1])
	if secondToThird < 1900*time.Millisecond || secondToThird > 2100*time.Millisecond {
		t.Errorf("Expected ~2s backoff between second and third attempt, got %v", secondToThird)
	}

	totalDuration := time.Since(start)
	if totalDuration < 3*time.Second {
		t.Errorf("Expected at least 3 seconds total, got %v", totalDuration)
	}
}
