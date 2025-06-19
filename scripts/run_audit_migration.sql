-- Run audit log migration
-- This script creates the audit log table and related indexes

\echo 'Creating audit log table and indexes...'

-- Read and execute the audit migration
\i migrations/002_create_audit_logs.sql

\echo 'Audit log migration completed successfully!'

-- Verify the table was created
SELECT 
    table_name, 
    column_name, 
    data_type, 
    is_nullable 
FROM information_schema.columns 
WHERE table_name = 'audit_logs' 
ORDER BY ordinal_position;

-- Show indexes
SELECT 
    indexname, 
    indexdef 
FROM pg_indexes 
WHERE tablename = 'audit_logs';

\echo 'Audit log table verification completed.' 