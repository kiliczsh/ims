// Package handlers provides HTTP request handlers for the IMS REST API.
// It includes handlers for audit logs, health checks, message management, and control operations.
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ims/internal/scheduler"

	"github.com/redis/go-redis/v9"
)

// Constants for health status values
const (
	HealthStatusHealthy       = "healthy"
	HealthStatusUnhealthy     = "unhealthy"
	HealthStatusConnected     = "connected"
	HealthStatusNotConfigured = "not_configured"
	HealthStatusStopped       = "stopped"
	HealthStatusRunning       = "running"
)

type HealthHandler struct {
	db        *sql.DB
	redis     *redis.Client
	scheduler *scheduler.Scheduler
}

func NewHealthHandler(db *sql.DB, redis *redis.Client, scheduler *scheduler.Scheduler) *HealthHandler {
	return &HealthHandler{
		db:        db,
		redis:     redis,
		scheduler: scheduler,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status" example:"healthy"`
	Timestamp time.Time              `json:"timestamp" example:"2023-12-01T10:00:00Z"`
	Scheduler map[string]interface{} `json:"scheduler"`
	Database  string                 `json:"database" example:"connected"`
	Redis     string                 `json:"redis" example:"connected"`
	Errors    []string               `json:"errors,omitempty"`
}

// Handle handles health check requests
// @Summary      Health Check
// @Description  Check the health status of the service including database, Redis, and scheduler
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
	}

	// Check scheduler status
	if h.scheduler != nil {
		running, startedAt := h.scheduler.GetStatus()
		response.Scheduler = map[string]interface{}{
			"running": running,
		}
		if startedAt != nil {
			response.Scheduler["started_at"] = startedAt
		}
	} else {
		response.Scheduler = map[string]interface{}{
			"running": false,
		}
	}

	// Check database connection
	if h.db != nil {
		if err := h.db.Ping(); err != nil {
			response.Status = HealthStatusUnhealthy
			response.Errors = append(response.Errors, "Database connection failed")
			response.Database = HealthStatusConnected
		} else {
			response.Database = HealthStatusConnected
		}
	} else {
		response.Database = HealthStatusNotConfigured
	}

	// Check Redis connection if configured
	if h.redis != nil {
		if err := h.redis.Ping(context.Background()).Err(); err != nil {
			response.Status = HealthStatusUnhealthy
			response.Errors = append(response.Errors, "Redis connection failed")
			response.Redis = HealthStatusConnected
		} else {
			response.Redis = HealthStatusConnected
		}
	} else {
		response.Redis = HealthStatusNotConfigured
	}

	statusCode := http.StatusOK
	if response.Status == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
