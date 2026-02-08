-- Email accounts (IMAP/SMTP configuration per user)
CREATE TABLE email_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email_address VARCHAR(255) NOT NULL,
    
    -- IMAP settings
    imap_host VARCHAR(255) NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    imap_username VARCHAR(255) NOT NULL,
    imap_password TEXT NOT NULL, -- Encrypted
    imap_use_tls BOOLEAN DEFAULT TRUE,
    
    -- SMTP settings
    smtp_host VARCHAR(255) NOT NULL,
    smtp_port INTEGER NOT NULL DEFAULT 587,
    smtp_username VARCHAR(255) NOT NULL,
    smtp_password TEXT NOT NULL, -- Encrypted
    smtp_use_tls BOOLEAN DEFAULT TRUE,
    
    -- Sync state
    last_sync_at TIMESTAMP,
    sync_error TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Email folders (synced from IMAP)
CREATE TABLE email_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    remote_name VARCHAR(255) NOT NULL, -- IMAP folder name
    folder_type VARCHAR(50), -- inbox, sent, drafts, trash, spam, archive, custom
    unread_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    uidvalidity BIGINT, -- IMAP UIDVALIDITY
    uidnext BIGINT, -- IMAP UIDNEXT for sync
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(account_id, remote_name)
);

-- Cached emails
CREATE TABLE emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES email_folders(id) ON DELETE CASCADE,
    message_id VARCHAR(512), -- Email Message-ID header
    uid BIGINT NOT NULL, -- IMAP UID
    
    -- Headers
    subject TEXT,
    from_address TEXT NOT NULL,
    from_name TEXT,
    to_addresses JSONB DEFAULT '[]',
    cc_addresses JSONB DEFAULT '[]',
    bcc_addresses JSONB DEFAULT '[]',
    reply_to TEXT,
    in_reply_to VARCHAR(512),
    
    -- Content
    text_body TEXT,
    html_body TEXT,
    snippet TEXT, -- Preview text
    
    -- Flags
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_answered BOOLEAN DEFAULT FALSE,
    is_draft BOOLEAN DEFAULT FALSE,
    has_attachments BOOLEAN DEFAULT FALSE,
    
    -- Dates
    date TIMESTAMP NOT NULL,
    received_at TIMESTAMP DEFAULT NOW(),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(folder_id, uid)
);

-- Email attachments
CREATE TABLE email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(255),
    size BIGINT,
    content_id VARCHAR(255), -- For inline attachments
    is_inline BOOLEAN DEFAULT FALSE,
    storage_key VARCHAR(512), -- Path in MinIO if downloaded
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_email_accounts_user ON email_accounts(user_id);
CREATE INDEX idx_email_folders_account ON email_folders(account_id);
CREATE INDEX idx_emails_folder ON emails(folder_id);
CREATE INDEX idx_emails_account ON emails(account_id);
CREATE INDEX idx_emails_date ON emails(date DESC);
CREATE INDEX idx_emails_message_id ON emails(message_id);
CREATE INDEX idx_emails_is_read ON emails(folder_id, is_read);
CREATE INDEX idx_email_attachments_email ON email_attachments(email_id);

-- Full text search on emails
CREATE INDEX idx_emails_search ON emails USING gin(to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(text_body, '')));
