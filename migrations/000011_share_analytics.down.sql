-- Remove share analytics columns
DROP INDEX IF EXISTS idx_shares_owner_analytics;
ALTER TABLE shares DROP COLUMN IF EXISTS view_count;
ALTER TABLE shares DROP COLUMN IF EXISTS last_accessed_at;
