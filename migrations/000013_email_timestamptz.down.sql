-- Revert TIMESTAMPTZ columns back to TIMESTAMP

-- email_accounts
ALTER TABLE email_accounts ALTER COLUMN last_sync_at TYPE TIMESTAMP;
ALTER TABLE email_accounts ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE email_accounts ALTER COLUMN updated_at TYPE TIMESTAMP;

-- email_folders
ALTER TABLE email_folders ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE email_folders ALTER COLUMN updated_at TYPE TIMESTAMP;

-- emails
ALTER TABLE emails ALTER COLUMN date TYPE TIMESTAMP;
ALTER TABLE emails ALTER COLUMN received_at TYPE TIMESTAMP;
ALTER TABLE emails ALTER COLUMN created_at TYPE TIMESTAMP;
ALTER TABLE emails ALTER COLUMN updated_at TYPE TIMESTAMP;

-- email_attachments
ALTER TABLE email_attachments ALTER COLUMN created_at TYPE TIMESTAMP;
