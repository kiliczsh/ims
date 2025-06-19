// Package service contains business logic and use case implementations for the IMS application.
// It coordinates between repositories and provides audit logging, message processing, and webhook services.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"ims/internal/domain"
)

type WebhookClient struct {
	client     *http.Client
	url        string
	authKey    string
	maxRetries int
}

func NewWebhookClient(url, authKey string, timeout time.Duration, maxRetries int) *WebhookClient {
	return &WebhookClient{
		client: &http.Client{
			Timeout: timeout,
		},
		url:        url,
		authKey:    authKey,
		maxRetries: maxRetries,
	}
}

func (w *WebhookClient) Send(ctx context.Context, phoneNumber, content string) (*domain.WebhookResponse, error) {
	req := domain.WebhookRequest{
		To:      phoneNumber,
		Content: content,
	}

	var resp domain.WebhookResponse
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := w.doRequest(ctx, req, &resp)
		if err == nil {
			return &resp, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", w.maxRetries+1, lastErr)
}

func (w *WebhookClient) doRequest(ctx context.Context, req domain.WebhookRequest, resp *domain.WebhookResponse) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-ins-auth-key", w.authKey)

	httpResp, err := w.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", httpResp.StatusCode)
	}

	// Try to decode JSON response, but handle cases where the webhook doesn't return JSON
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		// If JSON decoding fails, create a mock response for webhook.site
		// Generate a unique message ID for tracking
		resp.Message = "Accepted"
		resp.MessageID = fmt.Sprintf("webhook-%d", time.Now().UnixNano())

		// Log the issue but don't fail the request
		log.Printf("Webhook returned non-JSON response, using mock response: %s", resp.MessageID)
	}

	return nil
}
