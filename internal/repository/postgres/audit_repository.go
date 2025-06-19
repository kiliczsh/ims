package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"ims/internal/domain"
	"ims/internal/repository"
)

type auditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) repository.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Log(ctx context.Context, auditLog *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, event_type, event_name, description, batch_id, message_id, request_id,
			http_method, endpoint, status_code, duration_ms, message_count, 
			success_count, failure_count, metadata, created_at
		) VALUES (
			:id, :event_type, :event_name, :description, :batch_id, :message_id, :request_id,
			:http_method, :endpoint, :status_code, :duration_ms, :message_count,
			:success_count, :failure_count, :metadata, :created_at
		)`

	// Convert metadata to JSON
	var metadataJSON interface{}
	if auditLog.Metadata != nil && len(auditLog.Metadata) > 0 {
		jsonBytes, err := json.Marshal(auditLog.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = jsonBytes
	} else {
		metadataJSON = nil
	}

	// Create a map for named parameters
	params := map[string]interface{}{
		"id":            auditLog.ID,
		"event_type":    auditLog.EventType,
		"event_name":    auditLog.EventName,
		"description":   auditLog.Description,
		"batch_id":      auditLog.BatchID,
		"message_id":    auditLog.MessageID,
		"request_id":    auditLog.RequestID,
		"http_method":   auditLog.HTTPMethod,
		"endpoint":      auditLog.Endpoint,
		"status_code":   auditLog.StatusCode,
		"duration_ms":   auditLog.DurationMs,
		"message_count": auditLog.MessageCount,
		"success_count": auditLog.SuccessCount,
		"failure_count": auditLog.FailureCount,
		"metadata":      metadataJSON,
		"created_at":    auditLog.CreatedAt,
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

func (r *auditRepository) LogBatch(ctx context.Context, auditLogs []*domain.AuditLog) error {
	if len(auditLogs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO audit_logs (
			id, event_type, event_name, description, batch_id, message_id, request_id,
			http_method, endpoint, status_code, duration_ms, message_count, 
			success_count, failure_count, metadata, created_at
		) VALUES (
			:id, :event_type, :event_name, :description, :batch_id, :message_id, :request_id,
			:http_method, :endpoint, :status_code, :duration_ms, :message_count,
			:success_count, :failure_count, :metadata, :created_at
		)`

	for _, auditLog := range auditLogs {
		// Convert metadata to JSON
		var metadataJSON interface{}
		if auditLog.Metadata != nil && len(auditLog.Metadata) > 0 {
			jsonBytes, err := json.Marshal(auditLog.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadataJSON = jsonBytes
		} else {
			metadataJSON = nil
		}

		params := map[string]interface{}{
			"id":            auditLog.ID,
			"event_type":    auditLog.EventType,
			"event_name":    auditLog.EventName,
			"description":   auditLog.Description,
			"batch_id":      auditLog.BatchID,
			"message_id":    auditLog.MessageID,
			"request_id":    auditLog.RequestID,
			"http_method":   auditLog.HTTPMethod,
			"endpoint":      auditLog.Endpoint,
			"status_code":   auditLog.StatusCode,
			"duration_ms":   auditLog.DurationMs,
			"message_count": auditLog.MessageCount,
			"success_count": auditLog.SuccessCount,
			"failure_count": auditLog.FailureCount,
			"metadata":      metadataJSON,
			"created_at":    auditLog.CreatedAt,
		}

		_, err = tx.NamedExecContext(ctx, query, params)
		if err != nil {
			return fmt.Errorf("failed to insert audit log: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *auditRepository) GetAuditLogs(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	query := `
		SELECT 
			id, event_type, event_name, description, batch_id, message_id, request_id,
			http_method, endpoint, status_code, duration_ms, message_count, 
			success_count, failure_count, metadata, created_at
		FROM audit_logs`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter != nil {
		if len(filter.EventTypes) > 0 {
			eventTypes := make([]string, len(filter.EventTypes))
			for i, et := range filter.EventTypes {
				eventTypes[i] = string(et)
			}
			conditions = append(conditions, fmt.Sprintf("event_type = ANY($%d)", argIndex))
			args = append(args, pq.Array(eventTypes))
			argIndex++
		}

		if filter.BatchID != nil {
			conditions = append(conditions, fmt.Sprintf("batch_id = $%d", argIndex))
			args = append(args, *filter.BatchID)
			argIndex++
		}

		if filter.MessageID != nil {
			conditions = append(conditions, fmt.Sprintf("message_id = $%d", argIndex))
			args = append(args, *filter.MessageID)
			argIndex++
		}

		if filter.RequestID != nil {
			conditions = append(conditions, fmt.Sprintf("request_id = $%d", argIndex))
			args = append(args, *filter.RequestID)
			argIndex++
		}

		if filter.Endpoint != nil {
			conditions = append(conditions, fmt.Sprintf("endpoint = $%d", argIndex))
			args = append(args, *filter.Endpoint)
			argIndex++
		}

		if filter.FromDate != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.FromDate)
			argIndex++
		}

		if filter.ToDate != nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
			args = append(args, *filter.ToDate)
			argIndex++
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []*domain.AuditLog
	for rows.Next() {
		auditLog := &domain.AuditLog{}
		var metadataJSON []byte

		err := rows.Scan(
			&auditLog.ID, &auditLog.EventType, &auditLog.EventName, &auditLog.Description,
			&auditLog.BatchID, &auditLog.MessageID, &auditLog.RequestID,
			&auditLog.HTTPMethod, &auditLog.Endpoint, &auditLog.StatusCode,
			&auditLog.DurationMs, &auditLog.MessageCount, &auditLog.SuccessCount,
			&auditLog.FailureCount, &metadataJSON, &auditLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Parse metadata JSON
		if metadataJSON != nil {
			err = json.Unmarshal(metadataJSON, &auditLog.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return auditLogs, nil
}

func (r *auditRepository) GetAuditLogByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	query := `
		SELECT 
			id, event_type, event_name, description, batch_id, message_id, request_id,
			http_method, endpoint, status_code, duration_ms, message_count, 
			success_count, failure_count, metadata, created_at
		FROM audit_logs 
		WHERE id = $1`

	auditLog := &domain.AuditLog{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&auditLog.ID, &auditLog.EventType, &auditLog.EventName, &auditLog.Description,
		&auditLog.BatchID, &auditLog.MessageID, &auditLog.RequestID,
		&auditLog.HTTPMethod, &auditLog.Endpoint, &auditLog.StatusCode,
		&auditLog.DurationMs, &auditLog.MessageCount, &auditLog.SuccessCount,
		&auditLog.FailureCount, &metadataJSON, &auditLog.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	// Parse metadata JSON
	if metadataJSON != nil {
		err = json.Unmarshal(metadataJSON, &auditLog.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return auditLog, nil
}

func (r *auditRepository) GetBatchAuditLogs(ctx context.Context, batchID string) ([]*domain.AuditLog, error) {
	filter := &domain.AuditLogFilter{
		BatchID: &uuid.UUID{},
	}

	// Parse the batchID string to UUID
	err := filter.BatchID.UnmarshalText([]byte(batchID))
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID format: %w", err)
	}

	return r.GetAuditLogs(ctx, filter)
}

func (r *auditRepository) GetMessageAuditLogs(ctx context.Context, messageID string) ([]*domain.AuditLog, error) {
	filter := &domain.AuditLogFilter{
		MessageID: &uuid.UUID{},
	}

	// Parse the messageID string to UUID
	err := filter.MessageID.UnmarshalText([]byte(messageID))
	if err != nil {
		return nil, fmt.Errorf("invalid message ID format: %w", err)
	}

	return r.GetAuditLogs(ctx, filter)
}

func (r *auditRepository) GetAuditLogStats(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error) {
	// Build base query for counting
	query := `
		SELECT 
			COUNT(*) as total_count,
			event_type,
			COUNT(*) as event_count,
			MAX(created_at) as last_event_time,
			AVG(duration_ms) as avg_duration
		FROM audit_logs`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter != nil {
		if len(filter.EventTypes) > 0 {
			eventTypes := make([]string, len(filter.EventTypes))
			for i, et := range filter.EventTypes {
				eventTypes[i] = string(et)
			}
			conditions = append(conditions, fmt.Sprintf("event_type = ANY($%d)", argIndex))
			args = append(args, pq.Array(eventTypes))
			argIndex++
		}

		if filter.FromDate != nil {
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, *filter.FromDate)
			argIndex++
		}

		if filter.ToDate != nil {
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
			args = append(args, *filter.ToDate)
			argIndex++
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY event_type"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log stats: %w", err)
	}
	defer rows.Close()

	stats := &domain.AuditLogStats{
		EventTypeCounts: make(map[domain.AuditEventType]int64),
	}

	var totalCount int64
	var lastTime *time.Time
	var totalDuration float64
	var countWithDuration int64

	for rows.Next() {
		var count int64
		var eventType domain.AuditEventType
		var eventTime *time.Time
		var avgDuration *float64

		err := rows.Scan(&count, &eventType, &count, &eventTime, &avgDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}

		stats.EventTypeCounts[eventType] = count
		totalCount += count

		if eventTime != nil && (lastTime == nil || eventTime.After(*lastTime)) {
			lastTime = eventTime
		}

		if avgDuration != nil {
			totalDuration += *avgDuration * float64(count)
			countWithDuration += count
		}
	}

	stats.TotalCount = totalCount

	if lastTime != nil {
		timeStr := lastTime.Format(time.RFC3339)
		stats.LastEventTime = &timeStr
	}

	if countWithDuration > 0 {
		avgDur := totalDuration / float64(countWithDuration)
		stats.AverageRequestDuration = &avgDur
	}

	return stats, nil
}

func (r *auditRepository) DeleteOldAuditLogs(ctx context.Context, days int) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
