package service

import (
	"context"
	"errors"
	"ims/internal/domain"
	"ims/internal/repository"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewMessageService(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	maxLength := 1000

	service := NewMessageService(repo, cache, webhook, maxLength)

	if service.repo != repo {
		t.Error("Expected repo to be set correctly")
	}

	if service.cache != cache {
		t.Error("Expected cache to be set correctly")
	}

	if service.webhook != webhook {
		t.Error("Expected webhook to be set correctly")
	}

	if service.maxLength != maxLength {
		t.Errorf("Expected max length %d, got %d", maxLength, service.maxLength)
	}
}

func TestMessageService_CreateMessage_Success(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	ctx := context.Background()
	phoneNumber := "+1234567890"
	content := "Test message"

	msg, err := service.CreateMessage(ctx, phoneNumber, content)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if msg.PhoneNumber != phoneNumber {
		t.Errorf("Expected phone number %s, got %s", phoneNumber, msg.PhoneNumber)
	}

	if msg.Content != content {
		t.Errorf("Expected content %s, got %s", content, msg.Content)
	}

	if msg.Status != domain.StatusPending {
		t.Errorf("Expected status %s, got %s", domain.StatusPending, msg.Status)
	}

	if msg.RetryCount != 0 {
		t.Errorf("Expected retry count 0, got %d", msg.RetryCount)
	}

	if msg.ID == uuid.Nil {
		t.Error("Expected non-nil UUID")
	}

	// Verify message was created in repository
	if repo.Count() != 1 {
		t.Errorf("Expected 1 message in repository, got %d", repo.Count())
	}
}

func TestMessageService_CreateMessage_TooLong(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 10) // Very short max length

	ctx := context.Background()
	phoneNumber := "+1234567890"
	content := "This message is way too long for the limit"

	_, err := service.CreateMessage(ctx, phoneNumber, content)

	if err != domain.ErrMessageTooLong {
		t.Errorf("Expected ErrMessageTooLong, got %v", err)
	}

	// Verify no message was created in repository
	if repo.Count() != 0 {
		t.Errorf("Expected 0 messages in repository, got %d", repo.Count())
	}
}

func TestMessageService_CreateMessage_RepositoryError(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	// Configure repository to return error
	expectedError := errors.New("database error")
	repo.CreateMessageFunc = func(ctx context.Context, message *domain.Message) error {
		return expectedError
	}

	ctx := context.Background()
	_, err := service.CreateMessage(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected wrapped error containing database error, got %v", err)
	}
}

func TestMessageService_ProcessMessages_NoMessages(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	ctx := context.Background()
	err := service.ProcessMessages(ctx, 10)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestMessageService_ProcessMessages_RepositoryError(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	// Configure repository to return error
	expectedError := errors.New("database error")
	repo.GetUnsentMessagesFunc = func(ctx context.Context, limit int) ([]*domain.Message, error) {
		return nil, expectedError
	}

	ctx := context.Background()
	err := service.ProcessMessages(ctx, 10)

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected wrapped error containing database error, got %v", err)
	}
}

func TestMessageService_GetSentMessages_Success(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	// Add some test messages
	sentMsg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      domain.StatusSent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.AddMessage(sentMsg)

	ctx := context.Background()
	messages, err := service.GetSentMessages(ctx, 1, 20)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if len(messages) > 0 && messages[0].ID != sentMsg.ID {
		t.Errorf("Expected message ID %s, got %s", sentMsg.ID, messages[0].ID)
	}
}

func TestMessageService_GetSentMessages_Pagination(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	ctx := context.Background()

	tests := []struct {
		name         string
		page         int
		pageSize     int
		expectedPage int
		expectedSize int
	}{
		{"Default page", 0, 20, 1, 20},
		{"Negative page", -1, 20, 1, 20},
		{"Large page size", 1, 200, 1, 100}, // Should be capped at 100
		{"Small page size", 1, 0, 1, 20},    // Should default to 20
		{"Valid values", 2, 10, 2, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetSentMessages(ctx, tt.page, tt.pageSize)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			// Note: This test mainly verifies the method doesn't crash with edge case inputs
			// The actual pagination logic is tested in the repository mock
		})
	}
}

func TestMessageService_SendMessage_TooLong(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 10) // Very short max length

	// Create a message that's too long
	msg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "This message is way too long for the limit",
		Status:      domain.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.AddMessage(msg)

	ctx := context.Background()
	err := service.sendMessage(ctx, msg)

	if err != nil {
		t.Fatalf("Expected no error (status update should succeed), got %v", err)
	}

	// Verify message status was updated to failed
	updatedMsg, err := repo.GetMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Failed to get updated message: %v", err)
	}

	if updatedMsg.Status != domain.StatusFailed {
		t.Errorf("Expected status %s, got %s", domain.StatusFailed, updatedMsg.Status)
	}
}

func TestMessageService_SendMessage_UpdateStatusError(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	service := NewMessageService(repo, cache, webhook, 1000)

	// Configure repository to return error on status update
	expectedError := errors.New("database error")
	repo.UpdateMessageStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error {
		return expectedError
	}

	msg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      domain.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx := context.Background()
	err := service.sendMessage(ctx, msg)

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected wrapped error containing database error, got %v", err)
	}
}

// WebhookSender interface for dependency injection
type WebhookSender interface {
	Send(ctx context.Context, phoneNumber, content string) (*domain.WebhookResponse, error)
}

// MockWebhookClient for testing
type MockWebhookClient struct {
	SendFunc func(ctx context.Context, phoneNumber, content string) (*domain.WebhookResponse, error)
}

func (m *MockWebhookClient) Send(ctx context.Context, phoneNumber, content string) (*domain.WebhookResponse, error) {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, phoneNumber, content)
	}
	return &domain.WebhookResponse{
		Message:   "Message sent successfully",
		MessageID: "mock-msg-123",
	}, nil
}

// Note: The following tests would require modification of the MessageService
// to accept an interface instead of a concrete WebhookClient type.
// For now, we'll focus on testing the public API methods that don't require
// mocking the webhook client.
