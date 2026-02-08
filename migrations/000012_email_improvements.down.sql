ALTER TABLE email_accounts DROP COLUMN IF EXISTS signature;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS send_delay;

DROP TABLE IF EXISTS email_drafts;

-- Restore original search index
DROP INDEX IF EXISTS idx_emails_fulltext;
CREATE INDEX idx_emails_fulltext ON emails USING GIN (
    to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(text_body, ''))
);
