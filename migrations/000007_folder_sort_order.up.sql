-- Add sort_order for custom folder ordering
ALTER TABLE email_folders ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT 0;

-- Create index for ordering
CREATE INDEX IF NOT EXISTS idx_email_folders_sort ON email_folders(account_id, sort_order);
