-- Remove 2FA columns from users table
DROP INDEX IF EXISTS idx_users_totp_enabled;

ALTER TABLE users 
DROP COLUMN IF EXISTS totp_secret,
DROP COLUMN IF EXISTS totp_enabled,
DROP COLUMN IF EXISTS backup_codes;
