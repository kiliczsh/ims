// Package repository defines interfaces and implementations for data access layer.
// It provides abstractions for message and audit log storage with support for PostgreSQL and Redis.
package repository

import (
	"context"
	"time"

	"ims/internal/domain"

	"github.com/google/uuid"
)

type MessageRepository interface {
	GetUnsentMessages(ctx context.Context, limit int) ([]*domain.Message, error)
	UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error
	GetSentMessages(ctx context.Context, offset, limit int) ([]*domain.Message, error)
	GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	CreateMessage(ctx context.Context, message *domain.Message) error
}

type CacheRepository interface {
	SetMessageCache(ctx context.Context, messageID string, data interface{}, ttl time.Duration) error
	GetMessageCache(ctx context.Context, messageID string) (interface{}, error)
}
