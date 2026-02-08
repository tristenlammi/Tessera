-- Remove sort_order
DROP INDEX IF EXISTS idx_email_folders_sort;
ALTER TABLE email_folders DROP COLUMN IF EXISTS sort_order;
