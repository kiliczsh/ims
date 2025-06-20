-- migrations/003_create_dead_letter_queue.sql

-- Add dead_letter status to the existing enum
ALTER TYPE message_status ADD VALUE 'dead_letter';

-- Create dead letter messages table
CREATE TABLE IF NOT EXISTS dead_letter_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id UUID NOT NULL REFERENCES messages(id),
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    retry_count INTEGER NOT NULL,
    failure_reason TEXT NOT NULL,
    last_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL,
    moved_to_dlq_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    webhook_response TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add fields to messages table for better retry tracking
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS failure_reason TEXT,
ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMP WITH TIME ZONE;

-- Indexes for performance
CREATE INDEX idx_dead_letter_messages_original_id ON dead_letter_messages(original_message_id);
CREATE INDEX idx_dead_letter_messages_moved_at ON dead_letter_messages(moved_to_dlq_at);
CREATE INDEX idx_messages_next_retry_at ON messages(next_retry_at);
CREATE INDEX idx_messages_retry_count ON messages(retry_count);

-- Function to update updated_at timestamp for dead letter table
CREATE TRIGGER update_dead_letter_messages_updated_at BEFORE UPDATE
    ON dead_letter_messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column(); 