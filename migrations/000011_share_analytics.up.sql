-- Add share analytics columns
ALTER TABLE shares ADD COLUMN IF NOT EXISTS view_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE shares ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMP WITH TIME ZONE;

-- Create index for analytics queries
CREATE INDEX IF NOT EXISTS idx_shares_owner_analytics ON shares(owner_id, created_at);
