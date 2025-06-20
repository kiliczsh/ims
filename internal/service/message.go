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
	unsentMessages, err := s.repo.GetUnsentMessages(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unsent messages: %w", err)
	}

	// Fetch retryable messages (failed messages ready for retry)
	retryableMessages, err := s.repo.GetRetryableMessages(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get retryable messages: %w", err)
	}

	// Combine both message types
	allMessages := append(unsentMessages, retryableMessages...)

	if len(allMessages) == 0 {
		log.Println("No pending or retryable messages to process")
		return nil
	}

	log.Printf("Processing %d messages (%d new, %d retries)", len(allMessages), len(unsentMessages), len(retryableMessages))

	// Process each message
	for _, msg := range allMessages {
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
		// Move directly to dead letter queue for validation failures
		return s.repo.MoveToDeadLetterQueue(ctx, msg, "Message content exceeds maximum length", nil)
	}

	// Update status to sending
	if err := s.repo.UpdateMessageStatus(ctx, msg.ID, domain.StatusSending, nil); err != nil {
		return fmt.Errorf("failed to update message status to sending: %w", err)
	}

	log.Printf("Sending message %s to %s (attempt %d)", msg.ID, msg.PhoneNumber, msg.RetryCount+1)

	// Send via webhook
	resp, err := s.webhook.Send(ctx, msg.PhoneNumber, msg.Content)
	if err != nil {
		return s.handleSendFailure(ctx, msg, err, nil)
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

// handleSendFailure implements exponential backoff retry logic and dead letter queue
func (s *MessageService) handleSendFailure(ctx context.Context, msg *domain.Message, sendErr error, webhookResponse *string) error {
	const maxRetries = 5 // Maximum retry attempts before moving to DLQ

	newRetryCount := msg.RetryCount + 1
	failureReason := fmt.Sprintf("webhook failed: %v", sendErr)

	log.Printf("Message %s failed on attempt %d: %v", msg.ID, newRetryCount, sendErr)

	// Check if we've exceeded max retries
	if newRetryCount >= maxRetries {
		log.Printf("Message %s exceeded max retries (%d), moving to dead letter queue", msg.ID, maxRetries)
		return s.repo.MoveToDeadLetterQueue(ctx, msg,
			fmt.Sprintf("exceeded max retries (%d): %s", maxRetries, failureReason),
			webhookResponse)
	}

	// Calculate next retry time with exponential backoff
	// Retry delays: 1m, 4m, 9m, 16m, 25m
	backoffMinutes := newRetryCount * newRetryCount
	nextRetryAt := time.Now().Add(time.Duration(backoffMinutes) * time.Minute)

	log.Printf("Message %s will be retried in %d minutes at %v", msg.ID, backoffMinutes, nextRetryAt.Format("15:04:05"))

	// Update message with retry information
	return s.repo.UpdateMessageRetry(ctx, msg.ID, newRetryCount, &nextRetryAt, &failureReason)
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

func (s *MessageService) GetDeadLetterMessages(ctx context.Context, page, pageSize int) ([]*domain.DeadLetterMessage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.repo.GetDeadLetterMessages(ctx, offset, pageSize)
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
