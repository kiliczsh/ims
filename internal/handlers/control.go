// Package handlers provides HTTP request handlers for the IMS REST API.
// It includes handlers for audit logs, health checks, message management, and control operations.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ims/internal/scheduler"
)

type ControlHandler struct {
	scheduler *scheduler.Scheduler
}

func NewControlHandler(scheduler *scheduler.Scheduler) *ControlHandler {
	return &ControlHandler{scheduler: scheduler}
}

// ControlRequest represents a scheduler control request
type ControlRequest struct {
	Action string `json:"action" example:"start" enums:"start,stop"` // "start" or "stop"
}

// ControlResponse represents a scheduler control response
type ControlResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Scheduler started successfully"`
	Status  struct {
		Running   bool       `json:"running" example:"true"`
		StartedAt *time.Time `json:"started_at,omitempty" example:"2023-12-01T10:00:00Z"`
	} `json:"status"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid action. Use 'start' or 'stop'"`
}

// Handle handles scheduler control requests
// @Summary      Control Scheduler
// @Description  Start or stop the message scheduler
// @Tags         scheduler
// @Accept       json
// @Produce      json
// @Param        request   body      ControlRequest  true  "Control action"
// @Success      200       {object}  ControlResponse
// @Failure      400       {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /control [post]
func (h *ControlHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var resp ControlResponse

	switch req.Action {
	case "start":
		if err := h.scheduler.Start(r.Context()); err != nil {
			resp.Success = false
			resp.Message = err.Error()
		} else {
			resp.Success = true
			resp.Message = "Scheduler started successfully"
		}
	case "stop":
		if err := h.scheduler.Stop(); err != nil {
			resp.Success = false
			resp.Message = err.Error()
		} else {
			resp.Success = true
			resp.Message = "Scheduler stopped successfully"
		}
	default:
		http.Error(w, "Invalid action. Use 'start' or 'stop'", http.StatusBadRequest)
		return
	}

	running, startedAt := h.scheduler.GetStatus()
	resp.Status.Running = running
	resp.Status.StartedAt = startedAt

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
