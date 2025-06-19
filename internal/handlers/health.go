package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"ims/internal/scheduler"

	"github.com/redis/go-redis/v9"
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
		Status:    "healthy",
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
			response.Database = "disconnected"
			response.Status = "unhealthy"
		} else {
			response.Database = "connected"
		}
	} else {
		response.Database = "not_configured"
		response.Status = "unhealthy"
	}

	// Check Redis connection (optional)
	if h.redis != nil {
		if _, err := h.redis.Ping(r.Context()).Result(); err != nil {
			response.Redis = "disconnected"
		} else {
			response.Redis = "connected"
		}
	} else {
		response.Redis = "not_configured"
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
