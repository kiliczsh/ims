package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestHealthHandler_Handle_MethodNotAllowed(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHealthHandler_Handle_BasicResponse(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	// With nil dependencies, the database will fail to ping, making it unhealthy
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	var response HealthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got %s", response.Status)
	}

	if response.Database != "not_configured" {
		t.Errorf("Expected database 'not_configured', got %s", response.Database)
	}

	if response.Redis != "not_configured" {
		t.Errorf("Expected Redis 'not_configured', got %s", response.Redis)
	}

	// Scheduler should have a running field
	if _, exists := response.Scheduler["running"]; !exists {
		t.Error("Expected scheduler to have 'running' field")
	}
}

func TestHealthHandler_Handle_ContentType(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
	}
}

func TestHealthHandler_Handle_TimestampPresent(t *testing.T) {
	handler := NewHealthHandler(nil, nil, nil)

	beforeRequest := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.Handle(rr, req)
	afterRequest := time.Now()

	var response HealthResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Timestamp.Before(beforeRequest) || response.Timestamp.After(afterRequest) {
		t.Errorf("Expected timestamp to be between %v and %v, got %v",
			beforeRequest, afterRequest, response.Timestamp)
	}
}

func TestHealthResponse_JSONTags(t *testing.T) {
	startedAt := time.Now()
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Scheduler: map[string]interface{}{
			"running":    true,
			"started_at": &startedAt,
		},
		Database: "connected",
		Redis:    "connected",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Verify JSON contains expected fields
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	expectedFields := []string{"status", "timestamp", "scheduler", "database", "redis"}
	for _, field := range expectedFields {
		if _, exists := jsonMap[field]; !exists {
			t.Errorf("Expected field '%s' to be present in JSON", field)
		}
	}
}

func TestHealthResponse_Struct(t *testing.T) {
	// Test that the HealthResponse struct can be created and accessed
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Scheduler: map[string]interface{}{
			"running": true,
		},
		Database: "connected",
		Redis:    "connected",
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", response.Status)
	}

	if response.Database != "connected" {
		t.Errorf("Expected database 'connected', got %s", response.Database)
	}

	if response.Redis != "connected" {
		t.Errorf("Expected Redis 'connected', got %s", response.Redis)
	}

	if running, ok := response.Scheduler["running"].(bool); !ok || !running {
		t.Errorf("Expected scheduler running to be true, got %v", response.Scheduler["running"])
	}
}
