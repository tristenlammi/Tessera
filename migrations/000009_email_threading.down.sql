-- Remove threading columns from emails table
DROP INDEX IF EXISTS idx_emails_thread_id;
ALTER TABLE emails DROP COLUMN IF EXISTS references_header;
ALTER TABLE emails DROP COLUMN IF EXISTS thread_id;
