-- Settings table for storing application configuration
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert default module settings
INSERT INTO settings (key, value) VALUES 
    ('modules', '{"documents": false, "pdf": false, "tasks": false, "calendar": false, "contacts": false, "email": false}')
ON CONFLICT (key) DO NOTHING;
