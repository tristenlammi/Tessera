package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

type EmailRepository struct {
	db *pgxpool.Pool
}

func NewEmailRepository(db *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{db: db}
}

// sanitizeForDB ensures a string is valid UTF-8 for PostgreSQL
func sanitizeForDB(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var result []byte
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			if size == 0 {
				break
			}
			i += size
			continue
		}
		result = append(result, s[i:i+size]...)
		i += size
	}
	return string(result)
}

// ============ Email Accounts ============

func (r *EmailRepository) CreateAccount(ctx context.Context, account *models.EmailAccount) error {
	query := `
		INSERT INTO email_accounts (
			user_id, name, email_address,
			imap_host, imap_port, imap_username, imap_password, imap_use_tls,
			smtp_host, smtp_port, smtp_username, smtp_password, smtp_use_tls,
			is_default, signature, send_delay
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		account.UserID, account.Name, account.EmailAddress,
		account.IMAPHost, account.IMAPPort, account.IMAPUsername, account.IMAPPassword, account.IMAPUseTLS,
		account.SMTPHost, account.SMTPPort, account.SMTPUsername, account.SMTPPassword, account.SMTPUseTLS,
		account.IsDefault, account.Signature, account.SendDelay,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
}

func (r *EmailRepository) GetAccountByID(ctx context.Context, id string) (*models.EmailAccount, error) {
	query := `SELECT id, user_id, name, email_address,
		imap_host, imap_port, imap_username, imap_password, imap_use_tls,
		smtp_host, smtp_port, smtp_username, smtp_password, smtp_use_tls,
		last_sync_at, sync_error, is_default,
		COALESCE(signature, '') as signature, COALESCE(send_delay, 0) as send_delay,
		created_at, updated_at
		FROM email_accounts WHERE id = $1`
	var account models.EmailAccount
	err := r.db.QueryRow(ctx, query, id).Scan(
		&account.ID, &account.UserID, &account.Name, &account.EmailAddress,
		&account.IMAPHost, &account.IMAPPort, &account.IMAPUsername, &account.IMAPPassword, &account.IMAPUseTLS,
		&account.SMTPHost, &account.SMTPPort, &account.SMTPUsername, &account.SMTPPassword, &account.SMTPUseTLS,
		&account.LastSyncAt, &account.SyncError, &account.IsDefault,
		&account.Signature, &account.SendDelay,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *EmailRepository) GetAllAccounts(ctx context.Context) ([]models.EmailAccount, error) {
	query := `SELECT id, user_id, name, email_address,
		imap_host, imap_port, imap_username, imap_password, imap_use_tls,
		smtp_host, smtp_port, smtp_username, smtp_password, smtp_use_tls,
		last_sync_at, sync_error, is_default,
		COALESCE(signature, '') as signature, COALESCE(send_delay, 0) as send_delay,
		created_at, updated_at
		FROM email_accounts ORDER BY last_sync_at ASC NULLS FIRST`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.EmailAccount
	for rows.Next() {
		var a models.EmailAccount
		err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &a.EmailAddress,
			&a.IMAPHost, &a.IMAPPort, &a.IMAPUsername, &a.IMAPPassword, &a.IMAPUseTLS,
			&a.SMTPHost, &a.SMTPPort, &a.SMTPUsername, &a.SMTPPassword, &a.SMTPUseTLS,
			&a.LastSyncAt, &a.SyncError, &a.IsDefault,
			&a.Signature, &a.SendDelay,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *EmailRepository) GetAccountsByUser(ctx context.Context, userID string) ([]models.EmailAccount, error) {
	query := `SELECT id, user_id, name, email_address,
		imap_host, imap_port, imap_username, imap_password, imap_use_tls,
		smtp_host, smtp_port, smtp_username, smtp_password, smtp_use_tls,
		last_sync_at, sync_error, is_default,
		COALESCE(signature, '') as signature, COALESCE(send_delay, 0) as send_delay,
		created_at, updated_at
		FROM email_accounts WHERE user_id = $1 ORDER BY is_default DESC, name`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.EmailAccount
	for rows.Next() {
		var a models.EmailAccount
		err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &a.EmailAddress,
			&a.IMAPHost, &a.IMAPPort, &a.IMAPUsername, &a.IMAPPassword, &a.IMAPUseTLS,
			&a.SMTPHost, &a.SMTPPort, &a.SMTPUsername, &a.SMTPPassword, &a.SMTPUseTLS,
			&a.LastSyncAt, &a.SyncError, &a.IsDefault,
			&a.Signature, &a.SendDelay,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (r *EmailRepository) UpdateAccount(ctx context.Context, account *models.EmailAccount) error {
	query := `
		UPDATE email_accounts SET
			name = $2, email_address = $3,
			imap_host = $4, imap_port = $5, imap_username = $6, imap_password = $7, imap_use_tls = $8,
			smtp_host = $9, smtp_port = $10, smtp_username = $11, smtp_password = $12, smtp_use_tls = $13,
			is_default = $14, signature = $15, send_delay = $16, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		account.ID, account.Name, account.EmailAddress,
		account.IMAPHost, account.IMAPPort, account.IMAPUsername, account.IMAPPassword, account.IMAPUseTLS,
		account.SMTPHost, account.SMTPPort, account.SMTPUsername, account.SMTPPassword, account.SMTPUseTLS,
		account.IsDefault, account.Signature, account.SendDelay,
	)
	return err
}

func (r *EmailRepository) UpdateSyncStatus(ctx context.Context, accountID string, syncError *string) error {
	query := `UPDATE email_accounts SET last_sync_at = NOW(), sync_error = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, accountID, syncError)
	return err
}

func (r *EmailRepository) DeleteAccount(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_accounts WHERE id = $1`, id)
	return err
}

// ============ Email Folders ============

func (r *EmailRepository) UpsertFolder(ctx context.Context, folder *models.EmailFolder) error {
	query := `
		INSERT INTO email_folders (account_id, parent_id, name, remote_name, folder_type, delimiter, uidvalidity, uidnext)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (account_id, remote_name) DO UPDATE SET
			name = EXCLUDED.name, folder_type = EXCLUDED.folder_type,
			parent_id = EXCLUDED.parent_id, delimiter = EXCLUDED.delimiter,
			uidvalidity = EXCLUDED.uidvalidity, uidnext = EXCLUDED.uidnext,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		folder.AccountID, folder.ParentID, folder.Name, folder.RemoteName, folder.FolderType, folder.Delimiter,
		folder.UIDValidity, folder.UIDNext,
	).Scan(&folder.ID, &folder.CreatedAt, &folder.UpdatedAt)
}

func (r *EmailRepository) GetFoldersByAccount(ctx context.Context, accountID string) ([]models.EmailFolder, error) {
	query := `SELECT id, account_id, parent_id, name, remote_name, folder_type, delimiter, sort_order, unread_count, total_count, uidvalidity, uidnext, created_at, updated_at 
		FROM email_folders WHERE account_id = $1 ORDER BY 
		CASE folder_type 
			WHEN 'inbox' THEN 1 
			WHEN 'sent' THEN 2 
			WHEN 'drafts' THEN 3 
			WHEN 'trash' THEN 4 
			WHEN 'spam' THEN 5 
			ELSE 6 
		END, sort_order, name`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []models.EmailFolder
	for rows.Next() {
		var f models.EmailFolder
		err := rows.Scan(
			&f.ID, &f.AccountID, &f.ParentID, &f.Name, &f.RemoteName, &f.FolderType, &f.Delimiter, &f.SortOrder,
			&f.UnreadCount, &f.TotalCount, &f.UIDValidity, &f.UIDNext,
			&f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, nil
}

func (r *EmailRepository) GetFolderByID(ctx context.Context, id string) (*models.EmailFolder, error) {
	query := `SELECT id, account_id, parent_id, name, remote_name, folder_type, delimiter, sort_order, unread_count, total_count, uidvalidity, uidnext, created_at, updated_at FROM email_folders WHERE id = $1`
	var f models.EmailFolder
	err := r.db.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.AccountID, &f.ParentID, &f.Name, &f.RemoteName, &f.FolderType, &f.Delimiter, &f.SortOrder,
		&f.UnreadCount, &f.TotalCount, &f.UIDValidity, &f.UIDNext,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *EmailRepository) GetFolderByRemoteName(ctx context.Context, accountID, remoteName string) (*models.EmailFolder, error) {
	query := `SELECT id, account_id, parent_id, name, remote_name, folder_type, delimiter, sort_order, unread_count, total_count, uidvalidity, uidnext, created_at, updated_at FROM email_folders WHERE account_id = $1 AND remote_name = $2`
	var f models.EmailFolder
	err := r.db.QueryRow(ctx, query, accountID, remoteName).Scan(
		&f.ID, &f.AccountID, &f.ParentID, &f.Name, &f.RemoteName, &f.FolderType, &f.Delimiter, &f.SortOrder,
		&f.UnreadCount, &f.TotalCount, &f.UIDValidity, &f.UIDNext,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *EmailRepository) UpdateFolderCounts(ctx context.Context, folderID string) error {
	// Check if this is inbox - if so, exclude emails that are also in custom folders
	var folderType *string
	r.db.QueryRow(ctx, `SELECT folder_type FROM email_folders WHERE id = $1`, folderID).Scan(&folderType)

	var query string
	if folderType != nil && *folderType == "inbox" {
		query = `
			UPDATE email_folders SET 
				total_count = (
					SELECT COUNT(*) FROM emails e 
					WHERE e.folder_id = $1
					AND NOT EXISTS (
						SELECT 1 FROM emails e2
						JOIN email_folders f ON e2.folder_id = f.id
						WHERE e2.message_id = e.message_id
						AND e2.id != e.id
						AND f.folder_type = 'custom'
					)
				),
				unread_count = (
					SELECT COUNT(*) FROM emails e 
					WHERE e.folder_id = $1 AND e.is_read = false
					AND NOT EXISTS (
						SELECT 1 FROM emails e2
						JOIN email_folders f ON e2.folder_id = f.id
						WHERE e2.message_id = e.message_id
						AND e2.id != e.id
						AND f.folder_type = 'custom'
					)
				),
				updated_at = NOW()
			WHERE id = $1`
	} else {
		query = `
			UPDATE email_folders SET 
				total_count = (SELECT COUNT(*) FROM emails WHERE folder_id = $1),
				unread_count = (SELECT COUNT(*) FROM emails WHERE folder_id = $1 AND is_read = false),
				updated_at = NOW()
			WHERE id = $1`
	}
	_, err := r.db.Exec(ctx, query, folderID)
	return err
}

func (r *EmailRepository) GetFolderUIDValidity(ctx context.Context, folderID string) (int64, error) {
	var validity *int64
	err := r.db.QueryRow(ctx, `SELECT uidvalidity FROM email_folders WHERE id = $1`, folderID).Scan(&validity)
	if err != nil || validity == nil {
		return 0, err
	}
	return *validity, nil
}

func (r *EmailRepository) GetFolderUIDNext(ctx context.Context, folderID string) (int64, error) {
	var uidNext *int64
	err := r.db.QueryRow(ctx, `SELECT uidnext FROM email_folders WHERE id = $1`, folderID).Scan(&uidNext)
	if err != nil || uidNext == nil {
		return 0, err
	}
	return *uidNext, nil
}

func (r *EmailRepository) UpdateFolderUIDState(ctx context.Context, folderID string, uidValidity, uidNext int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE email_folders SET uidvalidity = $2, uidnext = $3, updated_at = NOW() WHERE id = $1`,
		folderID, uidValidity, uidNext)
	return err
}

func (r *EmailRepository) DeleteEmailsByFolder(ctx context.Context, folderID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM emails WHERE folder_id = $1`, folderID)
	return err
}

func (r *EmailRepository) CreateFolder(ctx context.Context, folder *models.EmailFolder) error {
	query := `
		INSERT INTO email_folders (account_id, parent_id, name, remote_name, folder_type, delimiter, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE((SELECT MAX(sort_order) + 1 FROM email_folders WHERE account_id = $1), 0))
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		folder.AccountID, folder.ParentID, folder.Name, folder.RemoteName, folder.FolderType, folder.Delimiter,
	).Scan(&folder.ID, &folder.CreatedAt, &folder.UpdatedAt)
}

func (r *EmailRepository) UpdateFolder(ctx context.Context, folder *models.EmailFolder) error {
	query := `
		UPDATE email_folders SET 
			name = $2, parent_id = $3, sort_order = $4, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, folder.ID, folder.Name, folder.ParentID, folder.SortOrder)
	return err
}

func (r *EmailRepository) DeleteFolder(ctx context.Context, folderID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_folders WHERE id = $1`, folderID)
	return err
}

func (r *EmailRepository) UpdateFolderSortOrder(ctx context.Context, folderID string, sortOrder int) error {
	_, err := r.db.Exec(ctx, `UPDATE email_folders SET sort_order = $2, updated_at = NOW() WHERE id = $1`, folderID, sortOrder)
	return err
}

// IncrementFolderSortOrders increments sort_order for all folders at or after the given position
func (r *EmailRepository) IncrementFolderSortOrders(ctx context.Context, accountID string, parentID *string, fromSortOrder int) error {
	var query string
	if parentID == nil || *parentID == "" {
		query = `UPDATE email_folders SET sort_order = sort_order + 1, updated_at = NOW() 
			WHERE account_id = $1 AND parent_id IS NULL AND sort_order >= $2`
		_, err := r.db.Exec(ctx, query, accountID, fromSortOrder)
		return err
	}
	query = `UPDATE email_folders SET sort_order = sort_order + 1, updated_at = NOW() 
		WHERE account_id = $1 AND parent_id = $2 AND sort_order >= $3`
	_, err := r.db.Exec(ctx, query, accountID, *parentID, fromSortOrder)
	return err
}

// ============ Emails ============

func (r *EmailRepository) CreateEmail(ctx context.Context, email *models.Email) error {
	// Sanitize all address fields before JSON marshaling
	for i := range email.To {
		email.To[i].Name = sanitizeForDB(email.To[i].Name)
		email.To[i].Address = sanitizeForDB(email.To[i].Address)
	}
	for i := range email.CC {
		email.CC[i].Name = sanitizeForDB(email.CC[i].Name)
		email.CC[i].Address = sanitizeForDB(email.CC[i].Address)
	}
	for i := range email.BCC {
		email.BCC[i].Name = sanitizeForDB(email.BCC[i].Name)
		email.BCC[i].Address = sanitizeForDB(email.BCC[i].Address)
	}

	toJSON, _ := json.Marshal(email.To)
	ccJSON, _ := json.Marshal(email.CC)
	bccJSON, _ := json.Marshal(email.BCC)

	query := `
		INSERT INTO emails (
			account_id, folder_id, message_id, uid,
			subject, from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
			reply_to, in_reply_to, text_body, html_body, snippet,
			is_read, is_starred, is_answered, is_draft, has_attachments, date,
			thread_id, references_header
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		ON CONFLICT (folder_id, uid) DO NOTHING
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		email.AccountID, email.FolderID, sanitizeForDB(email.MessageID), email.UID,
		sanitizeForDB(email.Subject), sanitizeForDB(email.FromAddress), sanitizeForDB(email.FromName), toJSON, ccJSON, bccJSON,
		sanitizeForDB(email.ReplyTo), sanitizeForDB(email.InReplyTo), sanitizeForDB(email.TextBody), sanitizeForDB(email.HTMLBody), sanitizeForDB(email.Snippet),
		email.IsRead, email.IsStarred, email.IsAnswered, email.IsDraft, email.HasAttachments, email.Date,
		sanitizeForDB(email.ThreadID), sanitizeForDB(email.ReferencesHeader),
	).Scan(&email.ID, &email.CreatedAt, &email.UpdatedAt)

	// If ON CONFLICT DO NOTHING was triggered, we get pgx.ErrNoRows
	// That's fine, it just means the email already exists
	if err != nil && err.Error() == "no rows in result set" {
		return nil // Not an error, just a duplicate
	}
	return err
}

func (r *EmailRepository) GetAllEmailsForAccount(ctx context.Context, accountID string) ([]models.Email, error) {
	query := `SELECT id, account_id, folder_id, message_id, uid, subject, from_address, from_name, 
		to_addresses, cc_addresses, bcc_addresses, reply_to, in_reply_to, text_body, html_body, snippet,
		is_read, is_starred, is_answered, is_draft, has_attachments, date, received_at, created_at, updated_at
		FROM emails WHERE account_id = $1`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var e models.Email
		err := rows.Scan(
			&e.ID, &e.AccountID, &e.FolderID, &e.MessageID, &e.UID,
			&e.Subject, &e.FromAddress, &e.FromName, &e.ToAddresses, &e.CCAddresses, &e.BCCAddresses,
			&e.ReplyTo, &e.InReplyTo, &e.TextBody, &e.HTMLBody, &e.Snippet,
			&e.IsRead, &e.IsStarred, &e.IsAnswered, &e.IsDraft, &e.HasAttachments,
			&e.Date, &e.ReceivedAt, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

func (r *EmailRepository) GetEmailByID(ctx context.Context, id string) (*models.Email, error) {
	query := `SELECT id, account_id, folder_id, message_id, uid,
		subject, from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
		reply_to, in_reply_to, text_body, html_body, snippet,
		is_read, is_starred, is_answered, is_draft, has_attachments,
		date, received_at, created_at, updated_at,
		COALESCE(thread_id, '') as thread_id, COALESCE(references_header, '') as references_header
		FROM emails WHERE id = $1`
	var e models.Email
	err := r.db.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.AccountID, &e.FolderID, &e.MessageID, &e.UID,
		&e.Subject, &e.FromAddress, &e.FromName, &e.ToAddresses, &e.CCAddresses, &e.BCCAddresses,
		&e.ReplyTo, &e.InReplyTo, &e.TextBody, &e.HTMLBody, &e.Snippet,
		&e.IsRead, &e.IsStarred, &e.IsAnswered, &e.IsDraft, &e.HasAttachments,
		&e.Date, &e.ReceivedAt, &e.CreatedAt, &e.UpdatedAt,
		&e.ThreadID, &e.ReferencesHeader,
	)
	if err != nil {
		return nil, err
	}
	e.ParseAddresses()
	return &e, nil
}

func (r *EmailRepository) GetEmailByMessageID(ctx context.Context, accountID, messageID string) (*models.Email, error) {
	// Normalize the message ID (strip angle brackets if present)
	normalizedID := strings.Trim(messageID, "<>")

	query := `SELECT id, account_id, folder_id, message_id, uid, 
		subject, from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
		reply_to, in_reply_to, text_body, html_body, snippet,
		is_read, is_starred, is_answered, is_draft, has_attachments,
		date, received_at, created_at, updated_at, 
		COALESCE(thread_id, '') as thread_id, COALESCE(references_header, '') as references_header
		FROM emails 
		WHERE account_id = $1 AND (message_id = $2 OR message_id = $3 OR message_id = $4)
		LIMIT 1`

	var e models.Email
	err := r.db.QueryRow(ctx, query, accountID, messageID, normalizedID, "<"+normalizedID+">").Scan(
		&e.ID, &e.AccountID, &e.FolderID, &e.MessageID, &e.UID,
		&e.Subject, &e.FromAddress, &e.FromName, &e.ToAddresses, &e.CCAddresses, &e.BCCAddresses,
		&e.ReplyTo, &e.InReplyTo, &e.TextBody, &e.HTMLBody, &e.Snippet,
		&e.IsRead, &e.IsStarred, &e.IsAnswered, &e.IsDraft, &e.HasAttachments,
		&e.Date, &e.ReceivedAt, &e.CreatedAt, &e.UpdatedAt, &e.ThreadID, &e.ReferencesHeader,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EmailRepository) GetEmailsByFolder(ctx context.Context, folderID string, limit, offset int) ([]models.EmailListItem, error) {
	// First check if this is the inbox folder - if so, exclude emails that are also in custom folders
	var folderType *string
	r.db.QueryRow(ctx, `SELECT folder_type FROM email_folders WHERE id = $1`, folderID).Scan(&folderType)

	var query string
	if folderType != nil && *folderType == "inbox" {
		// For inbox, exclude emails that also exist in custom folders (by message_id)
		query = `
			SELECT e.id, e.subject, e.from_address, e.from_name, e.snippet, e.date, e.is_read, e.is_starred, e.has_attachments
			FROM emails e
			WHERE e.folder_id = $1
			AND NOT EXISTS (
				SELECT 1 FROM emails e2
				JOIN email_folders f ON e2.folder_id = f.id
				WHERE e2.message_id = e.message_id
				AND e2.id != e.id
				AND f.folder_type = 'custom'
			)
			ORDER BY e.date DESC
			LIMIT $2 OFFSET $3`
	} else {
		query = `
			SELECT id, subject, from_address, from_name, snippet, date, is_read, is_starred, has_attachments
			FROM emails WHERE folder_id = $1
			ORDER BY date DESC
			LIMIT $2 OFFSET $3`
	}

	rows, err := r.db.Query(ctx, query, folderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

func (r *EmailRepository) UpdateEmailFlags(ctx context.Context, id string, isRead, isStarred *bool) error {
	if isRead == nil && isStarred == nil {
		return nil
	}

	query := `UPDATE emails SET updated_at = NOW()`
	args := []interface{}{}
	argNum := 1

	if isRead != nil {
		query += fmt.Sprintf(`, is_read = $%d`, argNum)
		args = append(args, *isRead)
		argNum++
	}
	if isStarred != nil {
		query += fmt.Sprintf(`, is_starred = $%d`, argNum)
		args = append(args, *isStarred)
		argNum++
	}

	query += fmt.Sprintf(` WHERE id = $%d`, argNum)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *EmailRepository) MarkAsRead(ctx context.Context, id string, isRead bool) error {
	_, err := r.db.Exec(ctx, `UPDATE emails SET is_read = $2, updated_at = NOW() WHERE id = $1`, id, isRead)
	return err
}

func (r *EmailRepository) UpdateEmailBody(ctx context.Context, id, textBody, htmlBody, snippet string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE emails SET text_body = $2, html_body = $3, snippet = $4, updated_at = NOW() WHERE id = $1`,
		id, sanitizeForDB(textBody), sanitizeForDB(htmlBody), sanitizeForDB(snippet))
	return err
}

func (r *EmailRepository) MarkFolderAsRead(ctx context.Context, folderID string) (int64, error) {
	result, err := r.db.Exec(ctx, `UPDATE emails SET is_read = true, updated_at = NOW() WHERE folder_id = $1 AND is_read = false`, folderID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *EmailRepository) MarkAsStarred(ctx context.Context, id string, isStarred bool) error {
	_, err := r.db.Exec(ctx, `UPDATE emails SET is_starred = $2, updated_at = NOW() WHERE id = $1`, id, isStarred)
	return err
}

func (r *EmailRepository) MoveEmail(ctx context.Context, emailID, newFolderID string) error {
	_, err := r.db.Exec(ctx, `UPDATE emails SET folder_id = $2, updated_at = NOW() WHERE id = $1`, emailID, newFolderID)
	return err
}

func (r *EmailRepository) DeleteEmail(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM emails WHERE id = $1`, id)
	return err
}

func (r *EmailRepository) SearchEmails(ctx context.Context, accountID, searchQuery string, limit int) ([]models.EmailListItem, error) {
	query := `
		SELECT id, subject, from_address, from_name, snippet, date, is_read, is_starred, has_attachments
		FROM emails 
		WHERE account_id = $1 
		AND to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(text_body, '')) @@ plainto_tsquery('english', $2)
		ORDER BY date DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, accountID, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

// ============ Attachments ============

func (r *EmailRepository) CreateAttachment(ctx context.Context, att *models.EmailAttachment) error {
	query := `
		INSERT INTO email_attachments (email_id, filename, content_type, size, content_id, is_inline, storage_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.db.QueryRow(ctx, query,
		att.EmailID, att.Filename, att.ContentType, att.Size, att.ContentID, att.IsInline, att.StorageKey,
	).Scan(&att.ID, &att.CreatedAt)
}

func (r *EmailRepository) GetAttachmentsByEmail(ctx context.Context, emailID string) ([]models.EmailAttachment, error) {
	query := `SELECT * FROM email_attachments WHERE email_id = $1 ORDER BY filename`

	rows, err := r.db.Query(ctx, query, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []models.EmailAttachment
	for rows.Next() {
		var a models.EmailAttachment
		err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size, &a.ContentID, &a.IsInline, &a.StorageKey, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}

// GetAttachmentsByEmailIDs returns attachments for multiple emails at once, grouped by email ID
func (r *EmailRepository) GetAttachmentsByEmailIDs(ctx context.Context, emailIDs []string) (map[string][]models.EmailAttachment, error) {
	if len(emailIDs) == 0 {
		return make(map[string][]models.EmailAttachment), nil
	}

	query := `SELECT id, email_id, filename, content_type, size, content_id, is_inline, storage_key, created_at
		FROM email_attachments WHERE email_id = ANY($1) ORDER BY filename`

	rows, err := r.db.Query(ctx, query, emailIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.EmailAttachment)
	for rows.Next() {
		var a models.EmailAttachment
		err := rows.Scan(&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size, &a.ContentID, &a.IsInline, &a.StorageKey, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		result[a.EmailID] = append(result[a.EmailID], a)
	}
	return result, nil
}

func (r *EmailRepository) GetAttachmentByID(ctx context.Context, attachmentID string) (*models.EmailAttachment, error) {
	query := `SELECT id, email_id, filename, content_type, size, content_id, is_inline, storage_key, created_at 
		FROM email_attachments WHERE id = $1`

	var a models.EmailAttachment
	err := r.db.QueryRow(ctx, query, attachmentID).Scan(
		&a.ID, &a.EmailID, &a.Filename, &a.ContentType, &a.Size, &a.ContentID, &a.IsInline, &a.StorageKey, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *EmailRepository) UpdateAttachmentStorageKey(ctx context.Context, attachmentID, storageKey string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_attachments SET storage_key = $1 WHERE id = $2`, storageKey, attachmentID)
	return err
}

func (r *EmailRepository) GetHighestUID(ctx context.Context, folderID string) (int64, error) {
	var uid int64
	err := r.db.QueryRow(ctx, `SELECT COALESCE(MAX(uid), 0) FROM emails WHERE folder_id = $1`, folderID).Scan(&uid)
	return uid, err
}

func (r *EmailRepository) GetEmailCountByFolder(ctx context.Context, folderID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM emails WHERE folder_id = $1`, folderID).Scan(&count)
	return count, err
}

// ============ Labels ============

func (r *EmailRepository) CreateLabel(ctx context.Context, label *models.EmailLabel) error {
	query := `
		INSERT INTO email_labels (account_id, name, color, is_system)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		label.AccountID, label.Name, label.Color, label.IsSystem,
	).Scan(&label.ID, &label.CreatedAt, &label.UpdatedAt)
}

func (r *EmailRepository) GetLabelByID(ctx context.Context, id string) (*models.EmailLabel, error) {
	query := `SELECT id, account_id, name, color, is_system, created_at, updated_at FROM email_labels WHERE id = $1`
	var label models.EmailLabel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&label.ID, &label.AccountID, &label.Name, &label.Color, &label.IsSystem, &label.CreatedAt, &label.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &label, nil
}

func (r *EmailRepository) GetLabelsByAccount(ctx context.Context, accountID string) ([]models.EmailLabel, error) {
	query := `
		SELECT l.id, l.account_id, l.name, l.color, l.is_system, l.created_at, l.updated_at,
			   COALESCE(COUNT(ela.id), 0) as email_count
		FROM email_labels l
		LEFT JOIN email_label_assignments ela ON l.id = ela.label_id
		WHERE l.account_id = $1
		GROUP BY l.id
		ORDER BY l.is_system DESC, l.name`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.EmailLabel
	for rows.Next() {
		var l models.EmailLabel
		err := rows.Scan(&l.ID, &l.AccountID, &l.Name, &l.Color, &l.IsSystem, &l.CreatedAt, &l.UpdatedAt, &l.EmailCount)
		if err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func (r *EmailRepository) UpdateLabel(ctx context.Context, label *models.EmailLabel) error {
	query := `UPDATE email_labels SET name = $2, color = $3, updated_at = NOW() WHERE id = $1 AND is_system = FALSE`
	_, err := r.db.Exec(ctx, query, label.ID, label.Name, label.Color)
	return err
}

func (r *EmailRepository) DeleteLabel(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_labels WHERE id = $1 AND is_system = FALSE`, id)
	return err
}

func (r *EmailRepository) AssignLabelToEmail(ctx context.Context, emailID, labelID string) error {
	query := `INSERT INTO email_label_assignments (email_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, query, emailID, labelID)
	return err
}

func (r *EmailRepository) RemoveLabelFromEmail(ctx context.Context, emailID, labelID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_label_assignments WHERE email_id = $1 AND label_id = $2`, emailID, labelID)
	return err
}

func (r *EmailRepository) GetLabelsByEmail(ctx context.Context, emailID string) ([]models.EmailLabel, error) {
	query := `
		SELECT l.id, l.account_id, l.name, l.color, l.is_system, l.created_at, l.updated_at
		FROM email_labels l
		JOIN email_label_assignments ela ON l.id = ela.label_id
		WHERE ela.email_id = $1
		ORDER BY l.name`

	rows, err := r.db.Query(ctx, query, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.EmailLabel
	for rows.Next() {
		var l models.EmailLabel
		err := rows.Scan(&l.ID, &l.AccountID, &l.Name, &l.Color, &l.IsSystem, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func (r *EmailRepository) GetEmailsByLabel(ctx context.Context, labelID string, limit, offset int) ([]models.EmailListItem, error) {
	query := `
		SELECT e.id, e.subject, e.from_address, e.from_name, e.snippet, e.date, e.is_read, e.is_starred, e.has_attachments
		FROM emails e
		JOIN email_label_assignments ela ON e.id = ela.email_id
		WHERE ela.label_id = $1
		ORDER BY e.date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, labelID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

// ============ Special Email Queries ============

func (r *EmailRepository) GetStarredEmails(ctx context.Context, accountID string, limit, offset int) ([]models.EmailListItem, error) {
	query := `
		SELECT id, subject, from_address, from_name, snippet, date, is_read, is_starred, has_attachments
		FROM emails
		WHERE account_id = $1 AND is_starred = TRUE
		ORDER BY date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

func (r *EmailRepository) GetDraftEmails(ctx context.Context, accountID string, limit, offset int) ([]models.EmailListItem, error) {
	query := `
		SELECT id, subject, from_address, from_name, snippet, date, is_read, is_starred, has_attachments
		FROM emails
		WHERE account_id = $1 AND is_draft = TRUE
		ORDER BY date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

func (r *EmailRepository) GetStarredCount(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM emails WHERE account_id = $1 AND is_starred = TRUE`, accountID).Scan(&count)
	return count, err
}

func (r *EmailRepository) GetDraftCount(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM emails WHERE account_id = $1 AND is_draft = TRUE`, accountID).Scan(&count)
	return count, err
}

// ============ Email Rules ============

func (r *EmailRepository) CreateRule(ctx context.Context, rule *models.EmailRule) error {
	if err := rule.SerializeRuleJSON(); err != nil {
		return err
	}

	query := `
		INSERT INTO email_rules (account_id, name, is_enabled, priority, match_type, conditions, actions, stop_processing)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		rule.AccountID, rule.Name, rule.IsEnabled, rule.Priority, rule.MatchType,
		rule.ConditionsJSON, rule.ActionsJSON, rule.StopProcessing,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *EmailRepository) GetRuleByID(ctx context.Context, id string) (*models.EmailRule, error) {
	query := `SELECT id, account_id, name, is_enabled, priority, match_type, conditions, actions, stop_processing, created_at, updated_at 
			  FROM email_rules WHERE id = $1`
	var rule models.EmailRule
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.AccountID, &rule.Name, &rule.IsEnabled, &rule.Priority, &rule.MatchType,
		&rule.ConditionsJSON, &rule.ActionsJSON, &rule.StopProcessing, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rule.ParseRuleJSON()
	return &rule, nil
}

func (r *EmailRepository) GetRulesByAccount(ctx context.Context, accountID string) ([]models.EmailRule, error) {
	query := `
		SELECT id, account_id, name, is_enabled, priority, match_type, conditions, actions, stop_processing, created_at, updated_at
		FROM email_rules
		WHERE account_id = $1
		ORDER BY priority, name`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.EmailRule
	for rows.Next() {
		var r models.EmailRule
		err := rows.Scan(&r.ID, &r.AccountID, &r.Name, &r.IsEnabled, &r.Priority, &r.MatchType,
			&r.ConditionsJSON, &r.ActionsJSON, &r.StopProcessing, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		r.ParseRuleJSON()
		rules = append(rules, r)
	}
	return rules, nil
}

func (r *EmailRepository) GetEnabledRules(ctx context.Context, accountID string) ([]models.EmailRule, error) {
	query := `
		SELECT id, account_id, name, is_enabled, priority, match_type, conditions, actions, stop_processing, created_at, updated_at
		FROM email_rules
		WHERE account_id = $1 AND is_enabled = TRUE
		ORDER BY priority`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.EmailRule
	for rows.Next() {
		var r models.EmailRule
		err := rows.Scan(&r.ID, &r.AccountID, &r.Name, &r.IsEnabled, &r.Priority, &r.MatchType,
			&r.ConditionsJSON, &r.ActionsJSON, &r.StopProcessing, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		r.ParseRuleJSON()
		rules = append(rules, r)
	}
	return rules, nil
}

func (r *EmailRepository) UpdateRule(ctx context.Context, rule *models.EmailRule) error {
	if err := rule.SerializeRuleJSON(); err != nil {
		return err
	}

	query := `
		UPDATE email_rules SET 
			name = $2, is_enabled = $3, priority = $4, match_type = $5, 
			conditions = $6, actions = $7, stop_processing = $8, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		rule.ID, rule.Name, rule.IsEnabled, rule.Priority, rule.MatchType,
		rule.ConditionsJSON, rule.ActionsJSON, rule.StopProcessing,
	)
	return err
}

func (r *EmailRepository) DeleteRule(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_rules WHERE id = $1`, id)
	return err
}

// GetThreadsForFolder returns conversation threads for a folder, grouped by thread_id
func (r *EmailRepository) GetThreadsForFolder(ctx context.Context, folderID string, limit, offset int) ([]models.EmailThread, error) {
	// Get threads with aggregated data
	query := `
		WITH thread_data AS (
			SELECT 
				thread_id,
				COUNT(*) as email_count,
				SUM(CASE WHEN is_read = false THEN 1 ELSE 0 END) as unread_count,
				MAX(date) as latest_date,
				BOOL_OR(has_attachments) as has_attachments,
				BOOL_OR(is_starred) as is_starred
			FROM emails
			WHERE folder_id = $1 AND thread_id IS NOT NULL AND thread_id != ''
			GROUP BY thread_id
		),
		latest_emails AS (
			SELECT DISTINCT ON (e.thread_id)
				e.id, e.thread_id, e.subject, e.snippet, e.from_address, e.from_name, e.date,
				e.is_read, e.is_starred, e.has_attachments, e.to_addresses
			FROM emails e
			WHERE e.folder_id = $1 AND e.thread_id IS NOT NULL AND e.thread_id != ''
			ORDER BY e.thread_id, e.date DESC
		)
		SELECT 
			td.thread_id,
			le.subject,
			le.snippet,
			td.latest_date,
			td.email_count,
			td.unread_count,
			td.has_attachments,
			td.is_starred,
			le.id,
			le.from_address,
			le.from_name,
			le.is_read as latest_is_read,
			le.is_starred as latest_is_starred,
			le.has_attachments as latest_has_attachments
		FROM thread_data td
		JOIN latest_emails le ON td.thread_id = le.thread_id
		ORDER BY td.latest_date DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, folderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []models.EmailThread
	for rows.Next() {
		var t models.EmailThread
		var latestID, latestFromAddr, latestFromName string
		var latestIsRead, latestIsStarred, latestHasAttachments bool

		err := rows.Scan(
			&t.ThreadID, &t.Subject, &t.Snippet, &t.LatestDate,
			&t.EmailCount, &t.UnreadCount, &t.HasAttachments, &t.IsStarred,
			&latestID, &latestFromAddr, &latestFromName,
			&latestIsRead, &latestIsStarred, &latestHasAttachments,
		)
		if err != nil {
			return nil, err
		}

		t.LatestEmail = &models.EmailListItem{
			ID:             latestID,
			ThreadID:       t.ThreadID,
			Subject:        t.Subject,
			Snippet:        t.Snippet,
			FromAddress:    latestFromAddr,
			FromName:       latestFromName,
			Date:           t.LatestDate,
			IsRead:         latestIsRead,
			IsStarred:      latestIsStarred,
			HasAttachments: latestHasAttachments,
			ThreadCount:    t.EmailCount,
		}

		threads = append(threads, t)
	}
	return threads, nil
}

// GetEmailsByThread returns all emails in a thread, ordered by date
func (r *EmailRepository) GetEmailsByThread(ctx context.Context, threadID string) ([]models.EmailListItem, error) {
	query := `
		SELECT id, thread_id, subject, from_address, from_name, snippet, date, is_read, is_starred, has_attachments
		FROM emails
		WHERE thread_id = $1
		ORDER BY date ASC`

	rows, err := r.db.Query(ctx, query, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.ThreadID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

// GetFullEmailsByThread returns complete Email objects for all emails in a thread
func (r *EmailRepository) GetFullEmailsByThread(ctx context.Context, threadID string) ([]models.Email, error) {
	// Use DISTINCT ON (message_id) to deduplicate emails that exist in multiple folders
	// (e.g., same email in both Inbox and Sent). Prefer inbox copy via folder_type ordering.
	query := `SELECT id, account_id, folder_id, message_id, uid,
		subject, from_address, from_name, to_addresses, cc_addresses, bcc_addresses,
		reply_to, in_reply_to, text_body, html_body, snippet,
		is_read, is_starred, is_answered, is_draft, has_attachments,
		date, received_at, created_at, updated_at,
		thread_id, COALESCE(references_header, '') as references_header
		FROM (
			SELECT DISTINCT ON (message_id) e.*,
				CASE WHEN f.folder_type = 'inbox' THEN 0
					 WHEN f.folder_type = 'sent' THEN 1
					 ELSE 2 END as folder_priority
			FROM emails e
			JOIN email_folders f ON e.folder_id = f.id
			WHERE e.thread_id = $1
			ORDER BY e.message_id, folder_priority, e.date ASC
		) deduped
		ORDER BY date ASC`

	rows, err := r.db.Query(ctx, query, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var e models.Email
		err := rows.Scan(
			&e.ID, &e.AccountID, &e.FolderID, &e.MessageID, &e.UID,
			&e.Subject, &e.FromAddress, &e.FromName, &e.ToAddresses, &e.CCAddresses, &e.BCCAddresses,
			&e.ReplyTo, &e.InReplyTo, &e.TextBody, &e.HTMLBody, &e.Snippet,
			&e.IsRead, &e.IsStarred, &e.IsAnswered, &e.IsDraft, &e.HasAttachments,
			&e.Date, &e.ReceivedAt, &e.CreatedAt, &e.UpdatedAt,
			&e.ThreadID, &e.ReferencesHeader,
		)
		if err != nil {
			return nil, err
		}
		e.ParseAddresses()
		emails = append(emails, e)
	}
	return emails, nil
}

// GetThreadCountForFolder returns the total number of threads in a folder
func (r *EmailRepository) GetThreadCountForFolder(ctx context.Context, folderID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT thread_id) 
		FROM emails 
		WHERE folder_id = $1 AND thread_id IS NOT NULL AND thread_id != ''
	`, folderID).Scan(&count)
	return count, err
}

// UpdateThreadIDsForExisting updates thread_id for existing emails that don't have one
func (r *EmailRepository) UpdateThreadIDsForExisting(ctx context.Context, accountID string) error {
	// First, set thread_id = message_id for emails without in_reply_to (thread roots)
	_, err := r.db.Exec(ctx, `
		UPDATE emails 
		SET thread_id = TRIM(BOTH '<>' FROM message_id)
		WHERE account_id = $1 
		AND (thread_id IS NULL OR thread_id = '')
		AND (in_reply_to IS NULL OR in_reply_to = '')
		AND message_id IS NOT NULL AND message_id != ''
	`, accountID)
	if err != nil {
		return err
	}

	// Then, propagate thread_id to replies
	// This may need multiple passes for deep threads
	for i := 0; i < 10; i++ { // Max 10 levels deep
		result, err := r.db.Exec(ctx, `
			UPDATE emails e
			SET thread_id = parent.thread_id
			FROM emails parent
			WHERE e.account_id = $1
			AND (e.thread_id IS NULL OR e.thread_id = '')
			AND e.in_reply_to IS NOT NULL AND e.in_reply_to != ''
			AND (
				parent.message_id = e.in_reply_to 
				OR parent.message_id = '<' || e.in_reply_to || '>'
				OR TRIM(BOTH '<>' FROM parent.message_id) = TRIM(BOTH '<>' FROM e.in_reply_to)
			)
			AND parent.thread_id IS NOT NULL AND parent.thread_id != ''
		`, accountID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			break // No more updates needed
		}
	}

	return nil
}

// ============ Advanced Search ============

// AdvancedSearchEmails supports Gmail-style operators: from:, to:, subject:, has:attachment, before:, after:, label:, is:starred, is:unread
func (r *EmailRepository) AdvancedSearchEmails(ctx context.Context, accountID, rawQuery string, limit int) ([]models.EmailListItem, error) {
	conditions := []string{"e.account_id = $1"}
	args := []interface{}{accountID}
	argN := 2
	freeText := rawQuery

	// Parse structured operators
	operators := map[string]func(string){
		"from:": func(val string) {
			conditions = append(conditions, fmt.Sprintf("(e.from_address ILIKE $%d OR e.from_name ILIKE $%d)", argN, argN))
			args = append(args, "%"+val+"%")
			argN++
		},
		"to:": func(val string) {
			conditions = append(conditions, fmt.Sprintf("e.to_addresses::text ILIKE $%d", argN))
			args = append(args, "%"+val+"%")
			argN++
		},
		"subject:": func(val string) {
			conditions = append(conditions, fmt.Sprintf("e.subject ILIKE $%d", argN))
			args = append(args, "%"+val+"%")
			argN++
		},
		"has:attachment": func(_ string) {
			conditions = append(conditions, "e.has_attachments = TRUE")
		},
		"is:starred": func(_ string) {
			conditions = append(conditions, "e.is_starred = TRUE")
		},
		"is:unread": func(_ string) {
			conditions = append(conditions, "e.is_read = FALSE")
		},
		"is:read": func(_ string) {
			conditions = append(conditions, "e.is_read = TRUE")
		},
		"before:": func(val string) {
			t, err := time.Parse("2006-01-02", val)
			if err == nil {
				conditions = append(conditions, fmt.Sprintf("e.date < $%d", argN))
				args = append(args, t)
				argN++
			}
		},
		"after:": func(val string) {
			t, err := time.Parse("2006-01-02", val)
			if err == nil {
				conditions = append(conditions, fmt.Sprintf("e.date > $%d", argN))
				args = append(args, t)
				argN++
			}
		},
	}

	// Extract operators from query
	for prefix, handler := range operators {
		for strings.Contains(freeText, prefix) {
			idx := strings.Index(freeText, prefix)
			// Find the value after the operator
			rest := freeText[idx+len(prefix):]
			var val string
			if strings.HasPrefix(rest, "\"") {
				// Quoted value
				end := strings.Index(rest[1:], "\"")
				if end >= 0 {
					val = rest[1 : end+1]
					freeText = freeText[:idx] + freeText[idx+len(prefix)+end+2:]
				} else {
					val = strings.TrimSpace(rest[1:])
					freeText = freeText[:idx]
				}
			} else {
				// Unquoted: take until next space
				parts := strings.SplitN(rest, " ", 2)
				val = parts[0]
				if len(parts) > 1 {
					freeText = freeText[:idx] + parts[1]
				} else {
					freeText = freeText[:idx]
				}
			}
			if prefix == "has:attachment" || prefix == "is:starred" || prefix == "is:unread" || prefix == "is:read" {
				handler("")
			} else {
				handler(strings.TrimSpace(val))
			}
		}
	}

	// Any remaining text is full-text search
	freeText = strings.TrimSpace(freeText)
	if freeText != "" {
		conditions = append(conditions, fmt.Sprintf(
			"to_tsvector('english', coalesce(e.subject, '') || ' ' || coalesce(e.from_name, '') || ' ' || coalesce(e.from_address, '') || ' ' || coalesce(e.text_body, '')) @@ plainto_tsquery('english', $%d)", argN))
		args = append(args, freeText)
		argN++
	}

	query := fmt.Sprintf(`
		SELECT e.id, e.subject, e.from_address, e.from_name, e.snippet, e.date, e.is_read, e.is_starred, e.has_attachments
		FROM emails e
		WHERE %s
		ORDER BY e.date DESC
		LIMIT $%d`, strings.Join(conditions, " AND "), argN)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []models.EmailListItem
	for rows.Next() {
		var e models.EmailListItem
		err := rows.Scan(&e.ID, &e.Subject, &e.FromAddress, &e.FromName, &e.Snippet, &e.Date, &e.IsRead, &e.IsStarred, &e.HasAttachments)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

// ============ Batch Operations ============

func (r *EmailRepository) BatchMarkAsRead(ctx context.Context, emailIDs []string, isRead bool) (int64, error) {
	if len(emailIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(emailIDs))
	args := []interface{}{isRead}
	for i, id := range emailIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf("UPDATE emails SET is_read = $1, updated_at = NOW() WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *EmailRepository) BatchMarkAsStarred(ctx context.Context, emailIDs []string, isStarred bool) (int64, error) {
	if len(emailIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(emailIDs))
	args := []interface{}{isStarred}
	for i, id := range emailIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf("UPDATE emails SET is_starred = $1, updated_at = NOW() WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *EmailRepository) BatchMoveEmails(ctx context.Context, emailIDs []string, targetFolderID string) (int64, error) {
	if len(emailIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(emailIDs))
	args := []interface{}{targetFolderID}
	for i, id := range emailIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf("UPDATE emails SET folder_id = $1, updated_at = NOW() WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *EmailRepository) BatchDeleteEmails(ctx context.Context, emailIDs []string) (int64, error) {
	if len(emailIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(emailIDs))
	args := make([]interface{}, len(emailIDs))
	for i, id := range emailIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf("DELETE FROM emails WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *EmailRepository) BatchAssignLabel(ctx context.Context, emailIDs []string, labelID string) (int64, error) {
	if len(emailIDs) == 0 {
		return 0, nil
	}
	var count int64
	for _, emailID := range emailIDs {
		_, err := r.db.Exec(ctx,
			`INSERT INTO email_label_assignments (email_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			emailID, labelID)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// ============ Drafts (Compose Drafts) ============

func (r *EmailRepository) SaveDraft(ctx context.Context, draft *models.EmailDraft) error {
	toJSON, _ := json.Marshal(draft.To)
	ccJSON, _ := json.Marshal(draft.CC)
	bccJSON, _ := json.Marshal(draft.BCC)

	if draft.ID == "" {
		// Insert new draft
		query := `INSERT INTO email_drafts (account_id, to_addresses, cc_addresses, bcc_addresses, subject, body, is_html, reply_to_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at, updated_at`
		return r.db.QueryRow(ctx, query,
			draft.AccountID, toJSON, ccJSON, bccJSON, draft.Subject, draft.Body, draft.IsHTML, draft.ReplyToID,
		).Scan(&draft.ID, &draft.CreatedAt, &draft.UpdatedAt)
	}

	// Update existing draft
	query := `UPDATE email_drafts SET
		to_addresses = $2, cc_addresses = $3, bcc_addresses = $4,
		subject = $5, body = $6, is_html = $7, reply_to_id = $8, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query,
		draft.ID, toJSON, ccJSON, bccJSON, draft.Subject, draft.Body, draft.IsHTML, draft.ReplyToID,
	)
	return err
}

func (r *EmailRepository) GetDraftByID(ctx context.Context, draftID string) (*models.EmailDraft, error) {
	query := `SELECT id, account_id, to_addresses, cc_addresses, bcc_addresses, subject, body, is_html, reply_to_id, created_at, updated_at
		FROM email_drafts WHERE id = $1`
	var d models.EmailDraft
	var toJSON, ccJSON, bccJSON []byte
	err := r.db.QueryRow(ctx, query, draftID).Scan(
		&d.ID, &d.AccountID, &toJSON, &ccJSON, &bccJSON,
		&d.Subject, &d.Body, &d.IsHTML, &d.ReplyToID,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(toJSON, &d.To)
	json.Unmarshal(ccJSON, &d.CC)
	json.Unmarshal(bccJSON, &d.BCC)
	return &d, nil
}

func (r *EmailRepository) GetDraftsByAccount(ctx context.Context, accountID string) ([]models.EmailDraft, error) {
	query := `SELECT id, account_id, to_addresses, cc_addresses, bcc_addresses, subject, body, is_html, reply_to_id, created_at, updated_at
		FROM email_drafts WHERE account_id = $1 ORDER BY updated_at DESC`

	rows, err := r.db.Query(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drafts []models.EmailDraft
	for rows.Next() {
		var d models.EmailDraft
		var toJSON, ccJSON, bccJSON []byte
		err := rows.Scan(
			&d.ID, &d.AccountID, &toJSON, &ccJSON, &bccJSON,
			&d.Subject, &d.Body, &d.IsHTML, &d.ReplyToID,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(toJSON, &d.To)
		json.Unmarshal(ccJSON, &d.CC)
		json.Unmarshal(bccJSON, &d.BCC)
		drafts = append(drafts, d)
	}
	return drafts, nil
}

func (r *EmailRepository) DeleteDraft(ctx context.Context, draftID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM email_drafts WHERE id = $1`, draftID)
	return err
}

// UpdateAccountSignature updates just the signature field
func (r *EmailRepository) UpdateAccountSignature(ctx context.Context, accountID, signature string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_accounts SET signature = $2, updated_at = NOW() WHERE id = $1`, accountID, signature)
	return err
}

// UpdateAccountSendDelay updates just the send_delay field
func (r *EmailRepository) UpdateAccountSendDelay(ctx context.Context, accountID string, sendDelay int) error {
	_, err := r.db.Exec(ctx, `UPDATE email_accounts SET send_delay = $2, updated_at = NOW() WHERE id = $1`, accountID, sendDelay)
	return err
}
