-- Add parent_id for nested folder support
ALTER TABLE email_folders ADD COLUMN parent_id UUID REFERENCES email_folders(id) ON DELETE CASCADE;

-- Add index for parent_id lookups
CREATE INDEX idx_email_folders_parent_id ON email_folders(parent_id);

-- Add delimiter column to store IMAP folder hierarchy delimiter
ALTER TABLE email_folders ADD COLUMN delimiter VARCHAR(10);
