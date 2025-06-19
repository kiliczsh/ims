package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"ims/internal/domain"
	"ims/internal/service"
)

type MessageHandler struct {
	service *service.MessageService
}

func NewMessageHandler(service *service.MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

// SentMessagesResponse represents a paginated list of sent messages
type SentMessagesResponse struct {
	Messages []*domain.SentMessageResponse `json:"messages"`
	Page     int                           `json:"page" example:"1"`
	PageSize int                           `json:"page_size" example:"20"`
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
	json.NewEncoder(w).Encode(resp)
}
