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

// scanMessagesFromRows is a helper function to scan multiple messages from rows
func (r *messageRepository) scanMessagesFromRows(rows *sql.Rows) ([]*domain.Message, error) {
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
			&msg.LastRetryAt,
			&msg.NextRetryAt,
			&msg.FailureReason,
			&msg.CreatedAt,
			&msg.SentAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}

// scanSingleMessageFromRow is a helper function to scan a single message from a row
func (r *messageRepository) scanSingleMessageFromRow(row *sql.Row) (*domain.Message, error) {
	msg := &domain.Message{}
	err := row.Scan(
		&msg.ID,
		&msg.PhoneNumber,
		&msg.Content,
		&msg.Status,
		&msg.MessageID,
		&msg.RetryCount,
		&msg.LastRetryAt,
		&msg.NextRetryAt,
		&msg.FailureReason,
		&msg.CreatedAt,
		&msg.SentAt,
		&msg.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	return msg, nil
}

func (r *messageRepository) GetUnsentMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, last_retry_at, next_retry_at, failure_reason, created_at, sent_at, updated_at
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

	return r.scanMessagesFromRows(rows)
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
		SELECT id, phone_number, content, status, message_id, retry_count, last_retry_at, next_retry_at, failure_reason, created_at, sent_at, updated_at
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

	return r.scanMessagesFromRows(rows)
}

func (r *messageRepository) GetMessage(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, last_retry_at, next_retry_at, failure_reason, created_at, sent_at, updated_at
		FROM messages 
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanSingleMessageFromRow(row)
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

func (r *messageRepository) GetRetryableMessages(ctx context.Context, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, phone_number, content, status, message_id, retry_count, last_retry_at, next_retry_at, failure_reason, created_at, sent_at, updated_at
		FROM messages 
		WHERE status = 'failed' AND next_retry_at IS NOT NULL AND next_retry_at <= CURRENT_TIMESTAMP
		ORDER BY next_retry_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query retryable messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessagesFromRows(rows)
}

func (r *messageRepository) UpdateMessageRetry(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt *time.Time, failureReason *string) error {
	query := `
		UPDATE messages 
		SET retry_count = $1, last_retry_at = CURRENT_TIMESTAMP, next_retry_at = $2, failure_reason = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query, retryCount, nextRetryAt, failureReason, id)
	if err != nil {
		return fmt.Errorf("failed to update message retry: %w", err)
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

func (r *messageRepository) MoveToDeadLetterQueue(ctx context.Context, message *domain.Message, failureReason string, webhookResponse *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert into dead letter queue
	dlqQuery := `
		INSERT INTO dead_letter_messages (id, original_message_id, phone_number, content, retry_count, failure_reason, last_attempt_at, webhook_response)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	dlqID := uuid.New()
	lastAttemptAt := time.Now()
	if message.LastRetryAt != nil {
		lastAttemptAt = *message.LastRetryAt
	}

	_, err = tx.ExecContext(ctx, dlqQuery,
		dlqID,
		message.ID,
		message.PhoneNumber,
		message.Content,
		message.RetryCount,
		failureReason,
		lastAttemptAt,
		webhookResponse,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into dead letter queue: %w", err)
	}

	// Update original message status
	updateQuery := `
		UPDATE messages 
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	_, err = tx.ExecContext(ctx, updateQuery, domain.StatusDeadLetter, message.ID)
	if err != nil {
		return fmt.Errorf("failed to update message status to dead_letter: %w", err)
	}

	return tx.Commit()
}

func (r *messageRepository) GetDeadLetterMessages(ctx context.Context, offset, limit int) ([]*domain.DeadLetterMessage, error) {
	query := `
		SELECT id, original_message_id, phone_number, content, retry_count, failure_reason, last_attempt_at, moved_to_dlq_at, webhook_response, created_at
		FROM dead_letter_messages
		ORDER BY moved_to_dlq_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query dead letter messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.DeadLetterMessage
	for rows.Next() {
		msg := &domain.DeadLetterMessage{}
		err := rows.Scan(
			&msg.ID,
			&msg.OriginalMessageID,
			&msg.PhoneNumber,
			&msg.Content,
			&msg.RetryCount,
			&msg.FailureReason,
			&msg.LastAttemptAt,
			&msg.MovedToDLQAt,
			&msg.WebhookResponse,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dead letter message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
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
