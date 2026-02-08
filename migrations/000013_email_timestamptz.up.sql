-- Convert all email-related TIMESTAMP columns to TIMESTAMPTZ
-- PostgreSQL interprets existing values as being in the server timezone (UTC by default)

-- email_accounts
ALTER TABLE email_accounts ALTER COLUMN last_sync_at TYPE TIMESTAMPTZ;
ALTER TABLE email_accounts ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE email_accounts ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- email_folders
ALTER TABLE email_folders ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE email_folders ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- emails (the critical ones)
ALTER TABLE emails ALTER COLUMN date TYPE TIMESTAMPTZ;
ALTER TABLE emails ALTER COLUMN received_at TYPE TIMESTAMPTZ;
ALTER TABLE emails ALTER COLUMN created_at TYPE TIMESTAMPTZ;
ALTER TABLE emails ALTER COLUMN updated_at TYPE TIMESTAMPTZ;

-- email_attachments
ALTER TABLE email_attachments ALTER COLUMN created_at TYPE TIMESTAMPTZ;
