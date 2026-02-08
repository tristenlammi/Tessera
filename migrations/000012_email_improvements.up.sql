-- Add signature and send_delay to email_accounts
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS signature TEXT NOT NULL DEFAULT '';
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS send_delay INTEGER NOT NULL DEFAULT 0;

-- Create drafts table for auto-save
CREATE TABLE IF NOT EXISTS email_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    to_addresses TEXT NOT NULL DEFAULT '',
    cc_addresses TEXT NOT NULL DEFAULT '',
    bcc_addresses TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    is_html BOOLEAN NOT NULL DEFAULT FALSE,
    reply_to_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_drafts_account ON email_drafts(account_id);

-- Add from_address to search index for advanced search
DROP INDEX IF EXISTS idx_emails_fulltext;
CREATE INDEX idx_emails_fulltext ON emails USING GIN (
    to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(text_body, '') || ' ' || coalesce(from_address, '') || ' ' || coalesce(from_name, ''))
);
