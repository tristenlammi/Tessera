-- Add threading support to emails table
ALTER TABLE emails ADD COLUMN IF NOT EXISTS thread_id VARCHAR(512);
ALTER TABLE emails ADD COLUMN IF NOT EXISTS references_header TEXT;

-- Create index for efficient thread queries
CREATE INDEX IF NOT EXISTS idx_emails_thread_id ON emails(thread_id);

-- Update existing emails to use message_id as thread_id where in_reply_to is null
-- This sets the thread root
UPDATE emails SET thread_id = message_id WHERE thread_id IS NULL AND (in_reply_to IS NULL OR in_reply_to = '');

-- For replies, we'll need to update thread_id through the application
-- since we need to traverse the in_reply_to chain
