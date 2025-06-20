package service

import (
	"context"
	"errors"
	"ims/internal/domain"
	"ims/internal/queue"
	"ims/internal/repository"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockQueueManager implements the queue.QueueManager interface for testing
type MockQueueManager struct {
	mockQueue *MockMessageQueue
}

func NewMockQueueManager() *MockQueueManager {
	return &MockQueueManager{
		mockQueue: NewMockMessageQueue(),
	}
}

func (m *MockQueueManager) GetQueue() queue.MessageQueue {
	return m.mockQueue
}

func (m *MockQueueManager) IsRabbitMQEnabled() bool {
	return false // For tests, default to database queue
}

// MockMessageQueue implements the queue.MessageQueue interface for testing
type MockMessageQueue struct {
	PublishFunc func(ctx context.Context, message *domain.Message) error
	ConsumeFunc func(ctx context.Context, handler queue.MessageHandler) error
	CloseFunc   func() error
	messages    []*domain.Message
}

func NewMockMessageQueue() *MockMessageQueue {
	return &MockMessageQueue{
		messages: make([]*domain.Message, 0),
	}
}

func (m *MockMessageQueue) Publish(ctx context.Context, message *domain.Message) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, message)
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockMessageQueue) Consume(ctx context.Context, handler queue.MessageHandler) error {
	if m.ConsumeFunc != nil {
		return m.ConsumeFunc(ctx, handler)
	}
	return nil
}

func (m *MockMessageQueue) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockMessageQueue) GetQueueType() queue.QueueType {
	return queue.QueueTypeDatabase
}

func (m *MockMessageQueue) GetMessages() []*domain.Message {
	return m.messages
}

func TestNewMessageService(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	maxLength := 1000

	service := NewMessageService(repo, cache, webhook, queueManager, maxLength)

	if service.repo != repo {
		t.Error("Expected repo to be set correctly")
	}

	if service.cache != cache {
		t.Error("Expected cache to be set correctly")
	}

	if service.webhook != webhook {
		t.Error("Expected webhook to be set correctly")
	}

	if service.queueManager != queueManager {
		t.Error("Expected queueManager to be set correctly")
	}

	if service.maxLength != maxLength {
		t.Errorf("Expected max length %d, got %d", maxLength, service.maxLength)
	}
}

func TestMessageService_CreateMessage_Success(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

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

	// Verify message was published to queue
	mockQueue := queueManager.GetQueue().(*MockMessageQueue)
	if len(mockQueue.GetMessages()) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(mockQueue.GetMessages()))
	}
}

func TestMessageService_CreateMessage_TooLong(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 10) // Very short max length

	ctx := context.Background()
	phoneNumber := "+1234567890"
	content := "This message is way too long for the limit"

	_, err := service.CreateMessage(ctx, phoneNumber, content)

	if err != domain.ErrMessageTooLong {
		t.Errorf("Expected ErrMessageTooLong, got %v", err)
	}

	// Verify no message was published to queue
	mockQueue := queueManager.GetQueue().(*MockMessageQueue)
	if len(mockQueue.GetMessages()) != 0 {
		t.Errorf("Expected 0 messages in queue, got %d", len(mockQueue.GetMessages()))
	}
}

func TestMessageService_CreateMessage_RepositoryError(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

	// Configure queue to return error
	expectedError := errors.New("queue error")
	mockQueue := queueManager.GetQueue().(*MockMessageQueue)
	mockQueue.PublishFunc = func(ctx context.Context, message *domain.Message) error {
		return expectedError
	}

	ctx := context.Background()
	_, err := service.CreateMessage(ctx, "+1234567890", "Test message")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("Expected wrapped error containing queue error, got %v", err)
	}
}

func TestMessageService_ProcessMessages_NoMessages(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

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
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

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
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

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
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

	// Add multiple test messages
	for i := 0; i < 5; i++ {
		msg := &domain.Message{
			ID:          uuid.New(),
			PhoneNumber: "+1234567890",
			Content:     "Test message",
			Status:      domain.StatusSent,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		repo.AddMessage(msg)
	}

	ctx := context.Background()

	// Test first page
	messages, err := service.GetSentMessages(ctx, 1, 3)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages on first page, got %d", len(messages))
	}

	// Test second page
	messages, err = service.GetSentMessages(ctx, 2, 3)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages on second page, got %d", len(messages))
	}
}

func TestMessageService_SendMessage_TooLong(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 10) // Very short max length

	// Create a message that's too long
	msg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "This message is way too long for the limit",
		Status:      domain.StatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx := context.Background()
	err := service.sendMessage(ctx, msg)

	if err != nil {
		t.Errorf("Expected no error from sendMessage (should handle internally), got %v", err)
	}

	// Verify message was moved to dead letter queue
	deadLetterMessages, err := repo.GetDeadLetterMessages(ctx, 0, 10)
	if err != nil {
		t.Fatalf("Failed to get dead letter messages: %v", err)
	}

	if len(deadLetterMessages) != 1 {
		t.Errorf("Expected 1 message in dead letter queue, got %d", len(deadLetterMessages))
	}

	if len(deadLetterMessages) > 0 {
		dlMsg := deadLetterMessages[0]
		if dlMsg.OriginalMessageID != msg.ID {
			t.Errorf("Expected original message ID %s, got %s", msg.ID, dlMsg.OriginalMessageID)
		}
		if dlMsg.FailureReason != "Message content exceeds maximum length" {
			t.Errorf("Expected specific failure reason, got %s", dlMsg.FailureReason)
		}
	}
}

func TestMessageService_SendMessage_UpdateStatusError(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

	// Configure repository to return error when updating status
	expectedError := errors.New("database error")
	repo.UpdateMessageStatusFunc = func(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error {
		return expectedError
	}

	msg := &domain.Message{
		ID:          uuid.New(),
		PhoneNumber: "+1234567890",
		Content:     "Test message",
		Status:      domain.StatusPending,
		RetryCount:  0,
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

func TestMessageService_CreateMessage_InvalidPhoneNumber(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

	tests := []struct {
		name        string
		phoneNumber string
	}{
		{"Empty phone number", ""},
		{"Missing plus sign", "1234567890"},
		{"Only plus sign", "+"},
		{"Plus with no digits", "+abc"},
		{"Plus with single digit", "+1"},
		{"Too short", "+12"},
		{"Too long", "+123456789012345678"},
		{"Invalid characters", "+123-456-7890"},
		{"Spaces", "+123 456 7890"},
		{"Starting with zero", "+01234567890"},
		{"Just spaces", "   "},
		{"Tabs and newlines", "\t+1234567890\n"},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateMessage(ctx, tt.phoneNumber, "Test message")

			if err != domain.ErrInvalidPhoneNumber {
				t.Errorf("Expected ErrInvalidPhoneNumber, got %v", err)
			}

			// Verify no message was published to queue
			mockQueue := queueManager.GetQueue().(*MockMessageQueue)
			if len(mockQueue.GetMessages()) != 0 {
				t.Errorf("Expected 0 messages in queue, got %d", len(mockQueue.GetMessages()))
			}
		})
	}
}

func TestMessageService_CreateMessage_ValidPhoneNumber(t *testing.T) {
	repo := repository.NewMockMessageRepository()
	cache := repository.NewMockCacheRepository()
	webhook := NewWebhookClient("http://example.com", "test-key", 30*time.Second, 3)
	queueManager := NewMockQueueManager()
	service := NewMessageService(repo, cache, webhook, queueManager, 1000)

	tests := []struct {
		name        string
		phoneNumber string
		expected    string // Expected cleaned phone number
	}{
		{"US number", "+1234567890", "+1234567890"},
		{"International short", "+123456789", "+123456789"},
		{"International long", "+12345678901234", "+12345678901234"},
		{"Maximum length", "+123456789012345", "+123456789012345"},
		{"Trimmed spaces", "  +1234567890  ", "+1234567890"},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear queue before test
			queueManager.mockQueue.messages = make([]*domain.Message, 0)

			msg, err := service.CreateMessage(ctx, tt.phoneNumber, "Test message")

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}

			if msg.PhoneNumber != tt.expected {
				t.Errorf("Expected phone number %s, got %s", tt.expected, msg.PhoneNumber)
			}

			// Verify message was published to queue
			mockQueue := queueManager.GetQueue().(*MockMessageQueue)
			if len(mockQueue.GetMessages()) != 1 {
				t.Errorf("Expected 1 message in queue, got %d", len(mockQueue.GetMessages()))
			}
		})
	}
}

// Note: The following tests would require modification of the MessageService
// to accept an interface instead of a concrete WebhookClient type.
// For now, we'll focus on testing the public API methods that don't require
// mocking the webhook client.
