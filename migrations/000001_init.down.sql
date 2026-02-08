-- Drop triggers first
DROP TRIGGER IF EXISTS audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_modification();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS shares;
DROP TABLE IF EXISTS file_versions;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS users;
