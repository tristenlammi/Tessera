-- Remove added columns from audit_logs table
ALTER TABLE audit_logs 
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS error;
