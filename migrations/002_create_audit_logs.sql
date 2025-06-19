-- migrations/002_create_audit_logs.sql
-- Audit logging system for tracking batches, requests, and system events

CREATE TYPE audit_event_type AS ENUM (
    'batch_started',
    'batch_completed', 
    'batch_failed',
    'message_sent',
    'message_failed',
    'scheduler_started',
    'scheduler_stopped',
    'api_request',
    'webhook_request',
    'webhook_response'
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type audit_event_type NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Context information
    batch_id UUID,
    message_id UUID,
    request_id VARCHAR(100),
    
    -- Request/Response details
    http_method VARCHAR(10),
    endpoint VARCHAR(255),
    status_code INTEGER,
    
    -- Metrics
    duration_ms INTEGER,
    message_count INTEGER,
    success_count INTEGER,
    failure_count INTEGER,
    
    -- Additional data (JSON)
    metadata JSONB,
    
    -- Timing
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_batch_id ON audit_logs(batch_id);
CREATE INDEX idx_audit_logs_message_id ON audit_logs(message_id);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX idx_audit_logs_endpoint ON audit_logs(endpoint);

-- Composite indexes for common queries
CREATE INDEX idx_audit_logs_type_created ON audit_logs(event_type, created_at);
CREATE INDEX idx_audit_logs_batch_type ON audit_logs(batch_id, event_type) WHERE batch_id IS NOT NULL; 