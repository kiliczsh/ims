package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ims/internal/domain"
	"ims/internal/service"
)

type AuditHandler struct {
	auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// GetAuditLogs godoc
// @Summary Get audit logs
// @Description Retrieve audit logs with optional filtering
// @Tags audit
// @Accept json
// @Produce json
// @Param event_types query []string false "Filter by event types"
// @Param batch_id query string false "Filter by batch ID"
// @Param message_id query string false "Filter by message ID"
// @Param request_id query string false "Filter by request ID"
// @Param endpoint query string false "Filter by endpoint"
// @Param from_date query string false "Filter from date (RFC3339 format)"
// @Param to_date query string false "Filter to date (RFC3339 format)"
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} domain.AuditLog
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /audit [get]
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	filter := &domain.AuditLogFilter{}

	// Parse query parameters
	query := r.URL.Query()

	// Event types
	if eventTypes := query["event_types"]; len(eventTypes) > 0 {
		filter.EventTypes = make([]domain.AuditEventType, len(eventTypes))
		for i, et := range eventTypes {
			filter.EventTypes[i] = domain.AuditEventType(et)
		}
	}

	// Batch ID
	if batchIDStr := query.Get("batch_id"); batchIDStr != "" {
		batchID, err := uuid.Parse(batchIDStr)
		if err != nil {
			http.Error(w, "Invalid batch_id format", http.StatusBadRequest)
			return
		}
		filter.BatchID = &batchID
	}

	// Message ID
	if messageIDStr := query.Get("message_id"); messageIDStr != "" {
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			http.Error(w, "Invalid message_id format", http.StatusBadRequest)
			return
		}
		filter.MessageID = &messageID
	}

	// Request ID
	if requestID := query.Get("request_id"); requestID != "" {
		filter.RequestID = &requestID
	}

	// Endpoint
	if endpoint := query.Get("endpoint"); endpoint != "" {
		filter.Endpoint = &endpoint
	}

	// From date
	if fromDateStr := query.Get("from_date"); fromDateStr != "" {
		fromDate, err := time.Parse(time.RFC3339, fromDateStr)
		if err != nil {
			http.Error(w, "Invalid from_date format, use RFC3339", http.StatusBadRequest)
			return
		}
		filter.FromDate = &fromDate
	}

	// To date
	if toDateStr := query.Get("to_date"); toDateStr != "" {
		toDate, err := time.Parse(time.RFC3339, toDateStr)
		if err != nil {
			http.Error(w, "Invalid to_date format, use RFC3339", http.StatusBadRequest)
			return
		}
		filter.ToDate = &toDate
	}

	// Limit
	if limitStr := query.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		filter.Limit = limit
	}

	// Offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
			return
		}
		filter.Offset = offset
	}

	// Default limit if not specified
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	auditLogs, err := h.auditService.GetAuditLogs(r.Context(), filter)
	if err != nil {
		http.Error(w, "Failed to get audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLogs)
}

// GetBatchAuditLogs godoc
// @Summary Get batch audit logs
// @Description Retrieve all audit logs for a specific batch
// @Tags audit
// @Accept json
// @Produce json
// @Param batch_id path string true "Batch ID"
// @Success 200 {array} domain.AuditLog
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /audit/batch/{batch_id} [get]
func (h *AuditHandler) GetBatchAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Extract batch ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/audit/batch/")
	batchID := strings.TrimSuffix(path, "/")

	if batchID == "" {
		http.Error(w, "Batch ID is required", http.StatusBadRequest)
		return
	}

	auditLogs, err := h.auditService.GetBatchAuditLogs(r.Context(), batchID)
	if err != nil {
		http.Error(w, "Failed to get batch audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLogs)
}

// GetMessageAuditLogs godoc
// @Summary Get message audit logs
// @Description Retrieve all audit logs for a specific message
// @Tags audit
// @Accept json
// @Produce json
// @Param message_id path string true "Message ID"
// @Success 200 {array} domain.AuditLog
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /audit/message/{message_id} [get]
func (h *AuditHandler) GetMessageAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Extract message ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/audit/message/")
	messageID := strings.TrimSuffix(path, "/")

	if messageID == "" {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	auditLogs, err := h.auditService.GetMessageAuditLogs(r.Context(), messageID)
	if err != nil {
		http.Error(w, "Failed to get message audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditLogs)
}

// GetAuditLogStats godoc
// @Summary Get audit log statistics
// @Description Retrieve statistics about audit logs
// @Tags audit
// @Accept json
// @Produce json
// @Param event_types query []string false "Filter by event types"
// @Param from_date query string false "Filter from date (RFC3339 format)"
// @Param to_date query string false "Filter to date (RFC3339 format)"
// @Success 200 {object} domain.AuditLogStats
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /audit/stats [get]
func (h *AuditHandler) GetAuditLogStats(w http.ResponseWriter, r *http.Request) {
	filter := &domain.AuditLogFilter{}

	// Parse query parameters
	query := r.URL.Query()

	// Event types
	if eventTypes := query["event_types"]; len(eventTypes) > 0 {
		filter.EventTypes = make([]domain.AuditEventType, len(eventTypes))
		for i, et := range eventTypes {
			filter.EventTypes[i] = domain.AuditEventType(et)
		}
	}

	// From date
	if fromDateStr := query.Get("from_date"); fromDateStr != "" {
		fromDate, err := time.Parse(time.RFC3339, fromDateStr)
		if err != nil {
			http.Error(w, "Invalid from_date format, use RFC3339", http.StatusBadRequest)
			return
		}
		filter.FromDate = &fromDate
	}

	// To date
	if toDateStr := query.Get("to_date"); toDateStr != "" {
		toDate, err := time.Parse(time.RFC3339, toDateStr)
		if err != nil {
			http.Error(w, "Invalid to_date format, use RFC3339", http.StatusBadRequest)
			return
		}
		filter.ToDate = &toDate
	}

	stats, err := h.auditService.GetAuditLogStats(r.Context(), filter)
	if err != nil {
		http.Error(w, "Failed to get audit log stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// CleanupOldAuditLogs godoc
// @Summary Cleanup old audit logs
// @Description Delete audit logs older than specified days
// @Tags audit
// @Accept json
// @Produce json
// @Param days query int true "Number of days to keep"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /audit/cleanup [delete]
func (h *AuditHandler) CleanupOldAuditLogs(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		http.Error(w, "days parameter is required", http.StatusBadRequest)
		return
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		http.Error(w, "Invalid days parameter, must be a positive integer", http.StatusBadRequest)
		return
	}

	deletedCount, err := h.auditService.CleanupOldAuditLogs(r.Context(), days)
	if err != nil {
		http.Error(w, "Failed to cleanup audit logs", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"deleted_count": deletedCount,
		"message":       "Audit logs cleanup completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
