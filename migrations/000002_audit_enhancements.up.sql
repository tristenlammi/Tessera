-- Add missing columns to audit_logs table
ALTER TABLE audit_logs 
    ADD COLUMN IF NOT EXISTS event_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'success',
    ADD COLUMN IF NOT EXISTS error TEXT,
    ALTER COLUMN user_id DROP NOT NULL;

-- Create index for event_type
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);
