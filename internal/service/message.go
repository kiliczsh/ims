package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"ims/internal/domain"
	"ims/internal/repository"

	"github.com/google/uuid"
)

// phoneNumberRegex defines a basic pattern for phone number validation
// Accepts formats like: +1234567890, +12345678901, +123456789012, etc.
var phoneNumberRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// validatePhoneNumber performs basic validation on phone number format
func validatePhoneNumber(phoneNumber string) bool {
	trimmed := strings.TrimSpace(phoneNumber)
	return phoneNumberRegex.MatchString(trimmed)
}

type MessageService struct {
	repo      repository.MessageRepository
	cache     repository.CacheRepository
	webhook   *WebhookClient
	maxLength int
}

func NewMessageService(
	repo repository.MessageRepository,
	cache repository.CacheRepository,
	webhook *WebhookClient,
	maxLength int,
) *MessageService {
	return &MessageService{
		repo:      repo,
		cache:     cache,
		webhook:   webhook,
		maxLength: maxLength,
	}
}

func (s *MessageService) ProcessMessages(ctx context.Context, batchSize int) error {
	// Fetch unsent messages
	messages, err := s.repo.GetUnsentMessages(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unsent messages: %w", err)
	}

	if len(messages) == 0 {
		log.Println("No pending messages to process")
		return nil
	}

	log.Printf("Processing %d messages", len(messages))

	// Process each message
	for _, msg := range messages {
		if err := s.sendMessage(ctx, msg); err != nil {
			log.Printf("Failed to send message %s: %v", msg.ID, err)
			// Continue with other messages even if one fails
			continue
		}
	}

	return nil
}

func (s *MessageService) sendMessage(ctx context.Context, msg *domain.Message) error {
	// Validate message content length
	if len(msg.Content) > s.maxLength {
		log.Printf("Message %s exceeds maximum length (%d > %d)", msg.ID, len(msg.Content), s.maxLength)
		return s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusFailed, nil)
	}

	// Update status to sending
	if err := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusSending, nil); err != nil {
		return fmt.Errorf("failed to update message status to sending: %w", err)
	}

	log.Printf("Sending message %s to %s", msg.ID, msg.PhoneNumber)

	// Send via webhook
	resp, err := s.webhook.Send(ctx, msg.PhoneNumber, msg.Content)
	if err != nil {
		log.Printf("Failed to send webhook for message %s: %v", msg.ID, err)
		// Update status to failed
		if updateErr := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusFailed, nil); updateErr != nil {
			log.Printf("Failed to update message status to failed: %v", updateErr)
		}
		return err
	}

	log.Printf("Message %s sent successfully, webhook response ID: %s", msg.ID, resp.MessageID)

	// Update status to sent
	if err := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusSent, &resp.MessageID); err != nil {
		return fmt.Errorf("failed to update message status to sent: %w", err)
	}

	// Cache message data (bonus)
	if s.cache != nil {
		cacheData := map[string]interface{}{
			"message_id":   resp.MessageID,
			"sent_at":      time.Now(),
			"phone_number": msg.PhoneNumber,
			"status_code":  202,
			"response":     resp,
		}
		if err := s.cache.SetMessageCache(ctx, resp.MessageID, cacheData, 168*time.Hour); err != nil {
			log.Printf("Failed to cache message data: %v", err)
			// Don't fail the operation if caching fails
		}
	}

	return nil
}

func (s *MessageService) GetSentMessages(ctx context.Context, page, pageSize int) ([]*domain.Message, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.repo.GetSentMessages(ctx, offset, pageSize)
}

func (s *MessageService) CreateMessage(ctx context.Context, phoneNumber, content string) (*domain.Message, error) {
	// Validate phone number format
	if !validatePhoneNumber(phoneNumber) {
		return nil, domain.ErrInvalidPhoneNumber
	}

	// Validate content length
	if len(content) > s.maxLength {
		return nil, domain.ErrMessageTooLong
	}

	msg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: strings.TrimSpace(phoneNumber),
		Content:     content,
		Status:      domain.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return msg, nil
}
