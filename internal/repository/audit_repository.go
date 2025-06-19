package repository

import (
	"context"

	"ims/internal/domain"
)

type AuditRepository interface {
	// Log creates a new audit log entry
	Log(ctx context.Context, auditLog *domain.AuditLog) error

	// LogBatch creates multiple audit log entries in a single transaction
	LogBatch(ctx context.Context, auditLogs []*domain.AuditLog) error

	// GetAuditLogs retrieves audit logs based on filter criteria
	GetAuditLogs(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error)

	// GetAuditLogByID retrieves a specific audit log by ID
	GetAuditLogByID(ctx context.Context, id string) (*domain.AuditLog, error)

	// GetBatchAuditLogs retrieves all audit logs for a specific batch
	GetBatchAuditLogs(ctx context.Context, batchID string) ([]*domain.AuditLog, error)

	// GetMessageAuditLogs retrieves all audit logs for a specific message
	GetMessageAuditLogs(ctx context.Context, messageID string) ([]*domain.AuditLog, error)

	// GetAuditLogStats returns statistics about audit logs
	GetAuditLogStats(ctx context.Context, filter *domain.AuditLogFilter) (*domain.AuditLogStats, error)

	// DeleteOldAuditLogs removes audit logs older than specified days
	DeleteOldAuditLogs(ctx context.Context, days int) (int64, error)
}

// AuditLogStats represents statistics about audit logs
type AuditLogStats struct {
	TotalCount             int64                           `json:"total_count"`
	EventTypeCounts        map[domain.AuditEventType]int64 `json:"event_type_counts"`
	LastEventTime          *string                         `json:"last_event_time,omitempty"`
	AverageRequestDuration *float64                        `json:"average_request_duration,omitempty"`
}

// Add AuditLogStats to domain package
func init() {
	// This ensures the stats type is available in domain package if needed
}
