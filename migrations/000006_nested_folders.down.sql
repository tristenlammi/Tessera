-- Remove nested folder support
DROP INDEX IF EXISTS idx_email_folders_parent_id;
ALTER TABLE email_folders DROP COLUMN IF EXISTS parent_id;
ALTER TABLE email_folders DROP COLUMN IF EXISTS delimiter;
