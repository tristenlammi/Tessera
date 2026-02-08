-- Add 2FA columns to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS totp_secret TEXT,
ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS backup_codes TEXT[];

-- Index for faster 2FA checks during login
CREATE INDEX IF NOT EXISTS idx_users_totp_enabled ON users(totp_enabled) WHERE totp_enabled = TRUE;

COMMENT ON COLUMN users.totp_secret IS 'Encrypted TOTP secret for 2FA';
COMMENT ON COLUMN users.totp_enabled IS 'Whether 2FA is enabled for this user';
COMMENT ON COLUMN users.backup_codes IS 'Hashed backup codes for 2FA recovery';
