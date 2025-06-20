// Package handlers provides HTTP request handlers for the IMS REST API.
// It includes handlers for audit logs, health checks, message management, and control operations.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"ims/internal/domain"
	"ims/internal/service"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(service *service.MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

// CreateMessageRequest represents the request body for creating a new message
type CreateMessageRequest struct {
	PhoneNumber string `json:"phone_number" example:"+1234567890"`
	Content     string `json:"content" example:"Hello, this is a test message"`
}

// CreateMessageResponse represents the response for a successfully created message
type CreateMessageResponse struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	PhoneNumber string `json:"phone_number" example:"+1234567890"`
	Content     string `json:"content" example:"Hello, this is a test message"`
	Status      string `json:"status" example:"pending"`
	CreatedAt   string `json:"created_at" example:"2023-12-01T10:00:00Z"`
}

// CreateMessage creates a new message for processing
// @Summary      Create Message
// @Description  Create a new message that will be queued for sending
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        message body CreateMessageRequest true "Message details"
// @Success      201 {object} CreateMessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Security     ApiKeyAuth
// @Router       /messages [post]
func (h *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if strings.TrimSpace(req.PhoneNumber) == "" {
		http.Error(w, "Phone number is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	// Create the message
	message, err := h.service.CreateMessage(r.Context(), req.PhoneNumber, req.Content)
	if err != nil {
		log.Printf("Failed to create message: %v", err)
		if err == domain.ErrMessageTooLong {
			http.Error(w, "Message content exceeds maximum length", http.StatusBadRequest)
			return
		}
		if err == domain.ErrInvalidPhoneNumber {
			http.Error(w, "Invalid phone number format", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create message", http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := CreateMessageResponse{
		ID:          message.ID.String(),
		PhoneNumber: message.PhoneNumber,
		Content:     message.Content,
		Status:      string(message.Status),
		CreatedAt:   message.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetDeadLetterMessages retrieves dead letter messages with pagination
// @Summary      Get Dead Letter Messages
// @Description  Retrieve a paginated list of messages that failed permanently and were moved to the dead letter queue
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        page      query     int  false  "Page number (default: 1)"  minimum(1)
// @Param        page_size query     int  false  "Page size (default: 20, max: 100)"  minimum(1)  maximum(100)
// @Success      200       {object}  DeadLetterMessagesResponse
// @Failure      500       {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /messages/dead-letter [get]
func (h *MessageHandler) GetDeadLetterMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	messages, err := h.service.GetDeadLetterMessages(r.Context(), page, pageSize)
	if err != nil {
		http.Error(w, "Failed to retrieve dead letter messages", http.StatusInternalServerError)
		return
	}

	resp := DeadLetterMessagesResponse{
		Messages: messages,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SentMessagesResponse represents a paginated list of sent messages
type SentMessagesResponse struct {
	Messages []*domain.SentMessageResponse `json:"messages"`
	Page     int                           `json:"page" example:"1"`
	PageSize int                           `json:"page_size" example:"20"`
}

// DeadLetterMessagesResponse represents a paginated list of dead letter messages
type DeadLetterMessagesResponse struct {
	Messages []*domain.DeadLetterMessage `json:"messages"`
	Page     int                         `json:"page" example:"1"`
	PageSize int                         `json:"page_size" example:"20"`
}

// GetSentMessages retrieves sent messages with pagination
// @Summary      Get Sent Messages
// @Description  Retrieve a paginated list of successfully sent messages
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        page      query     int  false  "Page number (default: 1)"  minimum(1)
// @Param        page_size query     int  false  "Page size (default: 20, max: 100)"  minimum(1)  maximum(100)
// @Success      200       {object}  SentMessagesResponse
// @Failure      500       {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /messages/sent [get]
func (h *MessageHandler) GetSentMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	messages, err := h.service.GetSentMessages(r.Context(), page, pageSize)
	if err != nil {
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	sentMessages := make([]*domain.SentMessageResponse, 0, len(messages))
	for _, msg := range messages {
		if msg.Status == domain.StatusSent && msg.MessageID != nil && msg.SentAt != nil {
			sentMessages = append(sentMessages, &domain.SentMessageResponse{
				ID:          msg.ID,
				PhoneNumber: msg.PhoneNumber,
				Content:     msg.Content,
				MessageID:   *msg.MessageID,
				SentAt:      *msg.SentAt,
			})
		}
	}

	resp := SentMessagesResponse{
		Messages: sentMessages,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
