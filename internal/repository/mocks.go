package repository

import (
	"context"
	"sync"
	"time"

	"ims/internal/domain"

	"github.com/google/uuid"
)

// MockMessageRepository is a mock implementation of MessageRepository for testing
type MockMessageRepository struct {
	mu                 sync.RWMutex
	messages           map[uuid.UUID]*domain.Message
	deadLetterMessages []*domain.DeadLetterMessage

	// Control mock behavior
	GetUnsentMessagesFunc     func(ctx context.Context, limit int) ([]*domain.Message, error)
	GetRetryableMessagesFunc  func(ctx context.Context, limit int) ([]*domain.Message, error)
	UpdateMessageStatusFunc   func(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error
	UpdateMessageRetryFunc    func(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt *time.Time, failureReason *string) error
	GetSentMessagesFunc       func(ctx context.Context, offset, limit int) ([]*domain.Message, error)
	GetMessageFunc            func(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	CreateMessageFunc         func(ctx context.Context, message *domain.Message) error
	MoveToDeadLetterQueueFunc func(ctx context.Context, message *domain.Message, failureReason string, webhookResponse *string) error
	GetDeadLetterMessagesFunc func(ctx context.Context, offset, limit int) ([]*domain.DeadLetterMessage, error)
}

func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		messages: make(map[uuid.UUID]*domain.Message),
	}
}

func (m *MockMessageRepository) GetUnsentMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	if m.GetUnsentMessagesFunc != nil {
		return m.GetUnsentMessagesFunc(ctx, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var unsent []*domain.Message
	for _, msg := range m.messages {
		if msg.Status == domain.StatusPending && len(unsent) < limit {
			unsent = append(unsent, msg)
		}
	}
	return unsent, nil
}

func (m *MockMessageRepository) UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error {
	if m.UpdateMessageStatusFunc != nil {
		return m.UpdateMessageStatusFunc(ctx, id, status, messageID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	msg, exists := m.messages[id]
	if !exists {
		return domain.ErrMessageNotFound
	}

	msg.Status = status
	msg.MessageID = messageID
	msg.UpdatedAt = time.Now()
	if status == domain.StatusSent {
		now := time.Now()
		msg.SentAt = &now
	}

	return nil
}

func (m *MockMessageRepository) GetSentMessages(ctx context.Context, offset, limit int) ([]*domain.Message, error) {
	if m.GetSentMessagesFunc != nil {
		return m.GetSentMessagesFunc(ctx, offset, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var sent []*domain.Message
	for _, msg := range m.messages {
		if msg.Status == domain.StatusSent {
			sent = append(sent, msg)
		}
	}

	// Simple pagination
	start := offset
	end := offset + limit
	if start > len(sent) {
		return []*domain.Message{}, nil
	}
	if end > len(sent) {
		end = len(sent)
	}

	return sent[start:end], nil
}

func (m *MockMessageRepository) GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[id]
	if !exists {
		return nil, domain.ErrMessageNotFound
	}

	return msg, nil
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, message *domain.Message) error {
	if m.CreateMessageFunc != nil {
		return m.CreateMessageFunc(ctx, message)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[message.ID] = message
	return nil
}

// Helper methods for testing
func (m *MockMessageRepository) AddMessage(message *domain.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[message.ID] = message
}

func (m *MockMessageRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make(map[uuid.UUID]*domain.Message)
}

func (m *MockMessageRepository) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

func (m *MockMessageRepository) GetRetryableMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	if m.GetRetryableMessagesFunc != nil {
		return m.GetRetryableMessagesFunc(ctx, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var retryable []*domain.Message
	now := time.Now()
	for _, msg := range m.messages {
		if msg.Status == domain.StatusFailed && msg.NextRetryAt != nil && msg.NextRetryAt.Before(now) && len(retryable) < limit {
			retryable = append(retryable, msg)
		}
	}
	return retryable, nil
}

func (m *MockMessageRepository) UpdateMessageRetry(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt *time.Time, failureReason *string) error {
	if m.UpdateMessageRetryFunc != nil {
		return m.UpdateMessageRetryFunc(ctx, id, retryCount, nextRetryAt, failureReason)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	msg, exists := m.messages[id]
	if !exists {
		return domain.ErrMessageNotFound
	}

	msg.RetryCount = retryCount
	msg.NextRetryAt = nextRetryAt
	msg.FailureReason = failureReason
	msg.Status = domain.StatusFailed
	now := time.Now()
	msg.LastRetryAt = &now
	msg.UpdatedAt = now

	return nil
}

func (m *MockMessageRepository) MoveToDeadLetterQueue(ctx context.Context, message *domain.Message, failureReason string, webhookResponse *string) error {
	if m.MoveToDeadLetterQueueFunc != nil {
		return m.MoveToDeadLetterQueueFunc(ctx, message, failureReason, webhookResponse)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create dead letter message
	dlqMsg := &domain.DeadLetterMessage{
		ID:                uuid.New(),
		OriginalMessageID: message.ID,
		PhoneNumber:       message.PhoneNumber,
		Content:           message.Content,
		RetryCount:        message.RetryCount,
		FailureReason:     failureReason,
		LastAttemptAt:     time.Now(),
		MovedToDLQAt:      time.Now(),
		WebhookResponse:   webhookResponse,
		CreatedAt:         time.Now(),
	}

	m.deadLetterMessages = append(m.deadLetterMessages, dlqMsg)

	// Update original message status
	if msg, exists := m.messages[message.ID]; exists {
		msg.Status = domain.StatusDeadLetter
		msg.UpdatedAt = time.Now()
	}

	return nil
}

func (m *MockMessageRepository) GetDeadLetterMessages(ctx context.Context, offset, limit int) ([]*domain.DeadLetterMessage, error) {
	if m.GetDeadLetterMessagesFunc != nil {
		return m.GetDeadLetterMessagesFunc(ctx, offset, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple pagination
	start := offset
	end := offset + limit
	if start > len(m.deadLetterMessages) {
		return []*domain.DeadLetterMessage{}, nil
	}
	if end > len(m.deadLetterMessages) {
		end = len(m.deadLetterMessages)
	}

	return m.deadLetterMessages[start:end], nil
}

// MockCacheRepository is a mock implementation of CacheRepository for testing
type MockCacheRepository struct {
	mu    sync.RWMutex
	cache map[string]interface{}

	// Control mock behavior
	SetMessageCacheFunc func(ctx context.Context, messageID string, data interface{}, ttl time.Duration) error
	GetMessageCacheFunc func(ctx context.Context, messageID string) (interface{}, error)
}

func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{
		cache: make(map[string]interface{}),
	}
}

func (m *MockCacheRepository) SetMessageCache(ctx context.Context, messageID string, data interface{}, ttl time.Duration) error {
	if m.SetMessageCacheFunc != nil {
		return m.SetMessageCacheFunc(ctx, messageID, data, ttl)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[messageID] = data
	return nil
}

func (m *MockCacheRepository) GetMessageCache(ctx context.Context, messageID string) (interface{}, error) {
	if m.GetMessageCacheFunc != nil {
		return m.GetMessageCacheFunc(ctx, messageID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.cache[messageID]
	if !exists {
		return nil, domain.ErrMessageNotFound
	}
	return data, nil
}

// Helper methods for testing
func (m *MockCacheRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]interface{})
}

func (m *MockCacheRepository) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cache)
}

// MockAuditRepository is a mock implementation of AuditRepository for testing
type MockAuditRepository struct {
	mu   sync.RWMutex
	logs []*domain.AuditLog

	// Control mock behavior
	LogFunc                 func(ctx context.Context, auditLog *domain.AuditLog) error
	LogBatchFunc            func(ctx context.Context, auditLogs []*domain.AuditLog) error
	GetAuditLogsFunc        func(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error)
	GetAuditLogByIDFunc     func(ctx context.Context, id string) (*domain.AuditLog, error)
	GetBatchAuditLogsFunc   func(ctx context.Context, batchID string) ([]*domain.AuditLog, error)
	GetMessageAuditLogsFunc func(ctx context.Context, messageID string) ([]*domain.AuditLog, error)
	GetAuditLogStatsFunc    func(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error)
	DeleteOldAuditLogsFunc  func(ctx context.Context, days int) (int64, error)
}

func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		logs: make([]*domain.AuditLog, 0),
	}
}

func (m *MockAuditRepository) Log(ctx context.Context, auditLog *domain.AuditLog) error {
	if m.LogFunc != nil {
		return m.LogFunc(ctx, auditLog)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, auditLog)
	return nil
}

func (m *MockAuditRepository) LogBatch(ctx context.Context, auditLogs []*domain.AuditLog) error {
	if m.LogBatchFunc != nil {
		return m.LogBatchFunc(ctx, auditLogs)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, auditLogs...)
	return nil
}

func (m *MockAuditRepository) GetAuditLogs(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	if m.GetAuditLogsFunc != nil {
		return m.GetAuditLogsFunc(ctx, filter)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple filtering logic for mock
	var filtered []*domain.AuditLog
	for _, log := range m.logs {
		if m.matchesFilter(log, filter) {
			filtered = append(filtered, log)
		}
	}

	// Apply pagination
	if filter != nil {
		start := filter.Offset
		end := start + filter.Limit
		if start > len(filtered) {
			return []*domain.AuditLog{}, nil
		}
		if end > len(filtered) {
			end = len(filtered)
		}
		if filter.Limit > 0 {
			filtered = filtered[start:end]
		}
	}

	return filtered, nil
}

func (m *MockAuditRepository) GetAuditLogByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	if m.GetAuditLogByIDFunc != nil {
		return m.GetAuditLogByIDFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, log := range m.logs {
		if log.ID.String() == id {
			return log, nil
		}
	}

	return nil, domain.ErrMessageNotFound
}

func (m *MockAuditRepository) GetBatchAuditLogs(ctx context.Context, batchID string) ([]*domain.AuditLog, error) {
	if m.GetBatchAuditLogsFunc != nil {
		return m.GetBatchAuditLogsFunc(ctx, batchID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []*domain.AuditLog
	for _, log := range m.logs {
		if log.BatchID != nil && log.BatchID.String() == batchID {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}

func (m *MockAuditRepository) GetMessageAuditLogs(ctx context.Context, messageID string) ([]*domain.AuditLog, error) {
	if m.GetMessageAuditLogsFunc != nil {
		return m.GetMessageAuditLogsFunc(ctx, messageID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []*domain.AuditLog
	for _, log := range m.logs {
		if log.MessageID != nil && log.MessageID.String() == messageID {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}

func (m *MockAuditRepository) GetAuditLogStats(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error) {
	if m.GetAuditLogStatsFunc != nil {
		return m.GetAuditLogStatsFunc(ctx, filter)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &domain.AuditLogStats{
		TotalCount:      int64(len(m.logs)),
		EventTypeCounts: make(map[domain.AuditEventType]int64),
	}

	for _, log := range m.logs {
		stats.EventTypeCounts[log.EventType]++
	}

	return stats, nil
}

func (m *MockAuditRepository) DeleteOldAuditLogs(ctx context.Context, days int) (int64, error) {
	if m.DeleteOldAuditLogsFunc != nil {
		return m.DeleteOldAuditLogsFunc(ctx, days)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	var kept []*domain.AuditLog
	deleted := int64(0)

	for _, log := range m.logs {
		if log.CreatedAt.After(cutoff) {
			kept = append(kept, log)
		} else {
			deleted++
		}
	}

	m.logs = kept
	return deleted, nil
}

// Helper methods for testing
func (m *MockAuditRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = make([]*domain.AuditLog, 0)
}

func (m *MockAuditRepository) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.logs)
}

func (m *MockAuditRepository) AddLog(log *domain.AuditLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, log)
}

func (m *MockAuditRepository) matchesFilter(log *domain.AuditLog, filter *domain.AuditLogFilter) bool {
	if filter == nil {
		return true
	}

	// Check event types
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if log.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check batch ID
	if filter.BatchID != nil {
		if log.BatchID == nil || *log.BatchID != *filter.BatchID {
			return false
		}
	}

	// Check message ID
	if filter.MessageID != nil {
		if log.MessageID == nil || *log.MessageID != *filter.MessageID {
			return false
		}
	}

	// Check request ID
	if filter.RequestID != nil {
		if log.RequestID == nil || *log.RequestID != *filter.RequestID {
			return false
		}
	}

	// Check endpoint
	if filter.Endpoint != nil {
		if log.Endpoint == nil || *log.Endpoint != *filter.Endpoint {
			return false
		}
	}

	// Check date range
	if filter.FromDate != nil && log.CreatedAt.Before(*filter.FromDate) {
		return false
	}

	if filter.ToDate != nil && log.CreatedAt.After(*filter.ToDate) {
		return false
	}

	return true
}
