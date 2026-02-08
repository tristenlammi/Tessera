DROP INDEX IF EXISTS idx_emails_search;
DROP INDEX IF EXISTS idx_email_attachments_email;
DROP INDEX IF EXISTS idx_emails_is_read;
DROP INDEX IF EXISTS idx_emails_message_id;
DROP INDEX IF EXISTS idx_emails_date;
DROP INDEX IF EXISTS idx_emails_account;
DROP INDEX IF EXISTS idx_emails_folder;
DROP INDEX IF EXISTS idx_email_folders_account;
DROP INDEX IF EXISTS idx_email_accounts_user;

DROP TABLE IF EXISTS email_attachments;
DROP TABLE IF EXISTS emails;
DROP TABLE IF EXISTS email_folders;
DROP TABLE IF EXISTS email_accounts;
