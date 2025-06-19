package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ims/internal/domain"
	"ims/internal/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type messageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) repository.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) GetUnsentMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, created_at, sent_at, updated_at
		FROM messages 
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unsent messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.PhoneNumber,
			&msg.Content,
			&msg.Status,
			&msg.MessageID,
			&msg.RetryCount,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) UpdateMessageStatus(ctx context.Context, id uuid.UUID, status domain.MessageStatus, messageID *string) error {
	var query string
	var args []interface{}

	if status == domain.StatusSent && messageID != nil {
		query = `
			UPDATE messages 
			SET status = $1, message_id = $2, sent_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`
		args = []interface{}{status, *messageID, id}
	} else {
		query = `
			UPDATE messages 
			SET status = $1, updated_at = CURRENT_TIMESTAMP
			WHERE id = $2
		`
		args = []interface{}{status, id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrMessageNotFound
	}

	return nil
}

func (r *messageRepository) GetSentMessages(ctx context.Context, offset, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, created_at, sent_at, updated_at
		FROM messages 
		WHERE status = 'sent'
		ORDER BY sent_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query sent messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.PhoneNumber,
			&msg.Content,
			&msg.Status,
			&msg.MessageID,
			&msg.RetryCount,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}

func (r *messageRepository) GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, created_at, sent_at, updated_at
		FROM messages 
		WHERE id = $1
	`

	msg := &domain.Message{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.PhoneNumber,
		&msg.Content,
		&msg.Status,
		&msg.MessageID,
		&msg.RetryCount,
		&msg.CreatedAt,
		&msg.SentAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return msg, nil
}

func (r *messageRepository) CreateMessage(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (id, phone_number, content, status, retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	if message.UpdatedAt.IsZero() {
		message.UpdatedAt = time.Now()
	}

	if message.Status == "" {
		message.Status = domain.StatusPending
	}

	_, err := r.db.ExecContext(ctx, query,
		message.ID,
		message.PhoneNumber,
		message.Content,
		message.Status,
		message.RetryCount,
		message.CreatedAt,
		message.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				return fmt.Errorf("message with this ID already exists: %w", err)
			}
		}
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

func NewDB(databaseURL string, maxConnections, maxIdleConnections int) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
