-- Drop indexes first
DROP INDEX IF EXISTS idx_emails_draft;
DROP INDEX IF EXISTS idx_emails_starred;
DROP INDEX IF EXISTS idx_email_rules_enabled;
DROP INDEX IF EXISTS idx_email_rules_account;
DROP INDEX IF EXISTS idx_email_label_assignments_label;
DROP INDEX IF EXISTS idx_email_label_assignments_email;
DROP INDEX IF EXISTS idx_email_labels_account;

-- Drop tables
DROP TABLE IF EXISTS email_rules;
DROP TABLE IF EXISTS email_label_assignments;
DROP TABLE IF EXISTS email_labels;
