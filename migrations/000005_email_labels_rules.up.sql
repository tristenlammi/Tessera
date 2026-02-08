-- Email labels (like Gmail labels - can be applied to multiple emails)
CREATE TABLE email_labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) DEFAULT '#6B7280', -- Hex color
    is_system BOOLEAN DEFAULT FALSE, -- System labels can't be deleted (inbox, sent, etc.)
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(account_id, name)
);

-- Junction table for email-label relationships (many-to-many)
CREATE TABLE email_label_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES email_labels(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(email_id, label_id)
);

-- Email rules for automatic labeling/moving
CREATE TABLE email_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0, -- Lower = higher priority
    
    -- Conditions (match any/all)
    match_type VARCHAR(10) DEFAULT 'any', -- 'any' or 'all'
    conditions JSONB NOT NULL DEFAULT '[]',
    -- Format: [{"field": "from"|"to"|"subject"|"body", "operator": "contains"|"equals"|"startswith"|"endswith", "value": "..."}]
    
    -- Actions
    actions JSONB NOT NULL DEFAULT '[]',
    -- Format: [{"type": "label"|"move"|"star"|"mark_read"|"archive"|"delete", "value": "label_id"|"folder_id"|null}]
    
    -- Stop processing more rules after this one matches
    stop_processing BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_email_labels_account ON email_labels(account_id);
CREATE INDEX idx_email_label_assignments_email ON email_label_assignments(email_id);
CREATE INDEX idx_email_label_assignments_label ON email_label_assignments(label_id);
CREATE INDEX idx_email_rules_account ON email_rules(account_id);
CREATE INDEX idx_email_rules_enabled ON email_rules(account_id, is_enabled, priority);

-- Index for starred emails lookup
CREATE INDEX idx_emails_starred ON emails(account_id, is_starred) WHERE is_starred = TRUE;

-- Index for draft emails lookup
CREATE INDEX idx_emails_draft ON emails(account_id, is_draft) WHERE is_draft = TRUE;
