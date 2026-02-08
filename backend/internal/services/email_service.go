package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog/log"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/security"
	"github.com/tessera/tessera/internal/storage"
)

type EmailService struct {
	repo      *repository.EmailRepository
	storage   storage.Storage
	encryptor *security.Encryptor
	imapPool  *IMAPPool
	// Pending sends for undo-send feature
	pendingSends     map[string]*models.PendingSend
	pendingSendsLock sync.Mutex
}

func NewEmailService(repo *repository.EmailRepository, store storage.Storage, encryptor *security.Encryptor) *EmailService {
	return &EmailService{
		repo:         repo,
		storage:      store,
		encryptor:    encryptor,
		imapPool:     NewIMAPPool(),
		pendingSends: make(map[string]*models.PendingSend),
	}
}

// encryptPassword encrypts a password for storage
func (s *EmailService) encryptPassword(password string) (string, error) {
	if s.encryptor == nil || password == "" {
		return password, nil // No encryption configured or empty password
	}

	encrypted, err := s.encryptor.Encrypt([]byte(password))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Store as base64 for safe storage in text column
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// decryptPassword decrypts a stored password
func (s *EmailService) decryptPassword(encryptedPassword string) (string, error) {
	if s.encryptor == nil || encryptedPassword == "" {
		return encryptedPassword, nil // No encryption configured or empty password
	}

	// Decode from base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		// Log warning but allow plaintext for migration
		log.Warn().Msg("Password not base64 encoded - may be plaintext from migration")
		return encryptedPassword, nil
	}

	decrypted, err := s.encryptor.Decrypt(encrypted)
	if err != nil {
		// Decryption failure is a security concern - log error and return failure
		// Don't return plaintext as fallback - this could leak encrypted data
		log.Error().Err(err).Msg("Password decryption failed - possible key mismatch or corruption")
		return "", fmt.Errorf("password decryption failed: %w", err)
	}

	return string(decrypted), nil
}

// encryptAccountPasswords encrypts the IMAP and SMTP passwords in an account
func (s *EmailService) encryptAccountPasswords(account *models.EmailAccount) error {
	var err error

	account.IMAPPassword, err = s.encryptPassword(account.IMAPPassword)
	if err != nil {
		return err
	}

	account.SMTPPassword, err = s.encryptPassword(account.SMTPPassword)
	if err != nil {
		return err
	}

	return nil
}

// decryptAccountPasswords decrypts the IMAP and SMTP passwords in an account
func (s *EmailService) decryptAccountPasswords(account *models.EmailAccount) error {
	var err error

	account.IMAPPassword, err = s.decryptPassword(account.IMAPPassword)
	if err != nil {
		return err
	}

	account.SMTPPassword, err = s.decryptPassword(account.SMTPPassword)
	if err != nil {
		return err
	}

	return nil
}

// ============ Account Management ============

func (s *EmailService) CreateAccount(ctx context.Context, account *models.EmailAccount) error {
	// Test connections with plain passwords first
	if err := s.testIMAPConnection(account); err != nil {
		return fmt.Errorf("IMAP connection failed: %w", err)
	}

	if err := s.testSMTPConnection(account); err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}

	// Encrypt passwords before storing
	if err := s.encryptAccountPasswords(account); err != nil {
		return fmt.Errorf("failed to encrypt passwords: %w", err)
	}

	return s.repo.CreateAccount(ctx, account)
}

func (s *EmailService) GetAccounts(ctx context.Context, userID string) ([]models.EmailAccount, error) {
	accounts, err := s.repo.GetAccountsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Decrypt passwords for use
	for i := range accounts {
		if err := s.decryptAccountPasswords(&accounts[i]); err != nil {
			return nil, err
		}
	}

	return accounts, nil
}

func (s *EmailService) GetAllAccounts(ctx context.Context) ([]models.EmailAccount, error) {
	accounts, err := s.repo.GetAllAccounts(ctx)
	if err != nil {
		return nil, err
	}

	// Decrypt passwords for use
	for i := range accounts {
		if err := s.decryptAccountPasswords(&accounts[i]); err != nil {
			return nil, err
		}
	}

	return accounts, nil
}

func (s *EmailService) GetAccount(ctx context.Context, accountID string) (*models.EmailAccount, error) {
	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Decrypt passwords for use
	if err := s.decryptAccountPasswords(account); err != nil {
		return nil, err
	}

	return account, nil
}

func (s *EmailService) UpdateAccount(ctx context.Context, account *models.EmailAccount) error {
	// Encrypt passwords before storing
	if err := s.encryptAccountPasswords(account); err != nil {
		return fmt.Errorf("failed to encrypt passwords: %w", err)
	}

	return s.repo.UpdateAccount(ctx, account)
}

func (s *EmailService) DeleteAccount(ctx context.Context, accountID string) error {
	return s.repo.DeleteAccount(ctx, accountID)
}

// ============ IMAP Operations ============

func (s *EmailService) testIMAPConnection(account *models.EmailAccount) error {
	client, err := s.connectIMAP(account)
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

func (s *EmailService) connectIMAP(account *models.EmailAccount) (*imapclient.Client, error) {
	return s.imapPool.Get(account)
}

// returnIMAP returns a connection to the pool for reuse
func (s *EmailService) returnIMAP(accountID string, client *imapclient.Client) {
	s.imapPool.Return(accountID, client)
}

func (s *EmailService) SyncAccount(ctx context.Context, accountID string) error {
	return s.SyncAccountWithProgress(ctx, accountID, nil)
}

type ProgressCallback func(progress map[string]interface{})

func (s *EmailService) SyncAccountWithProgress(ctx context.Context, accountID string, onProgress ProgressCallback) error {
	sendProgress := func(data map[string]interface{}) {
		if onProgress != nil {
			onProgress(data)
		}
	}

	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		sendProgress(map[string]interface{}{"type": "error", "message": err.Error()})
		return err
	}

	// Decrypt passwords for IMAP connection
	if err := s.decryptAccountPasswords(account); err != nil {
		sendProgress(map[string]interface{}{"type": "error", "message": "Failed to decrypt credentials"})
		return err
	}

	sendProgress(map[string]interface{}{"type": "progress", "message": "Connecting to email server..."})

	var client *imapclient.Client
	err = withRetry(3, func() error {
		var connectErr error
		client, connectErr = s.connectIMAP(account)
		return connectErr
	})
	if err != nil {
		errStr := err.Error()
		s.repo.UpdateSyncStatus(ctx, accountID, &errStr)
		sendProgress(map[string]interface{}{"type": "error", "message": errStr})
		return err
	}
	// Note: do NOT defer returnIMAP here. syncAllMailToInbox returns the client
	// to the pool mid-function and gets fresh connections for each batch.

	sendProgress(map[string]interface{}{"type": "progress", "message": "Setting up INBOX..."})

	// Create or get INBOX folder - all emails go here
	inboxFolder, err := s.ensureInboxFolder(ctx, account)
	if err != nil {
		s.returnIMAP(account.ID, client) // Return client to pool on failure
		errStr := err.Error()
		s.repo.UpdateSyncStatus(ctx, accountID, &errStr)
		sendProgress(map[string]interface{}{"type": "error", "message": errStr})
		return err
	}

	sendProgress(map[string]interface{}{
		"type":    "progress",
		"message": "Syncing all emails to INBOX...",
		"folder":  "INBOX",
	})

	// Sync ALL emails from Gmail's "All Mail" folder into local INBOX
	totalSyncedEmails, err := s.syncAllMailToInbox(ctx, client, account, inboxFolder)
	if err != nil {
		log.Error().Err(err).Str("account", accountID).Msg("Error syncing emails")
	}

	// Sync Sent folder separately
	sendProgress(map[string]interface{}{
		"type":    "progress",
		"message": "Syncing sent emails...",
		"folder":  "Sent",
	})

	sentSynced, err := s.syncSentFolder(ctx, account)
	if err != nil {
		log.Error().Err(err).Str("account", accountID).Msg("Error syncing sent folder")
	} else {
		totalSyncedEmails += sentSynced
	}

	s.repo.UpdateSyncStatus(ctx, accountID, nil)
	sendProgress(map[string]interface{}{
		"type":          "complete",
		"message":       fmt.Sprintf("Sync complete! Synced %d emails total", totalSyncedEmails),
		"synced_emails": totalSyncedEmails,
	})
	return nil
}

// ensureInboxFolder creates or retrieves the INBOX folder for the account
func (s *EmailService) ensureInboxFolder(ctx context.Context, account *models.EmailAccount) (*models.EmailFolder, error) {
	// Create all system folders
	systemFolders := []struct {
		name       string
		remoteName string
		folderType string
	}{
		{"Inbox", "INBOX", "inbox"},
		{"Sent", "[Gmail]/Sent Mail", "sent"},
		{"Drafts", "[Gmail]/Drafts", "drafts"},
		{"Trash", "[Gmail]/Trash", "trash"},
	}

	var inboxFolder *models.EmailFolder
	delimiter := "/"

	for _, sf := range systemFolders {
		folderType := sf.folderType
		folder := &models.EmailFolder{
			AccountID:  account.ID,
			Name:       sf.name,
			RemoteName: sf.remoteName,
			FolderType: &folderType,
			Delimiter:  &delimiter,
		}

		if err := s.repo.UpsertFolder(ctx, folder); err != nil {
			return nil, err
		}

		if sf.folderType == "inbox" {
			// Retrieve the folder with its ID
			folders, err := s.repo.GetFoldersByAccount(ctx, account.ID)
			if err != nil {
				return nil, err
			}
			for _, f := range folders {
				if f.FolderType != nil && *f.FolderType == "inbox" {
					inboxFolder = &f
					break
				}
			}
		}
	}

	if inboxFolder == nil {
		return nil, fmt.Errorf("failed to create inbox folder")
	}

	return inboxFolder, nil
}

// syncAllMailToInbox syncs all emails from Gmail's "All Mail" folder into the local INBOX
func (s *EmailService) syncAllMailToInbox(ctx context.Context, client *imapclient.Client, account *models.EmailAccount, inboxFolder *models.EmailFolder) (int, error) {
	// Gmail's "All Mail" folder contains all emails (sent, received, archived, etc.)
	// Try different possible names for "All Mail" folder
	allMailNames := []string{
		"[Gmail]/All Mail",
		"[Google Mail]/All Mail",
		"All Mail",
	}

	var selectedMailbox string
	var selectData *imap.SelectData
	var err error

	for _, name := range allMailNames {
		selectData, err = client.Select(name, nil).Wait()
		if err == nil {
			selectedMailbox = name
			log.Info().Str("folder", name).Uint32("messages", selectData.NumMessages).Msg("Found All Mail folder")
			break
		}
	}

	if selectedMailbox == "" {
		// Fallback to INBOX if All Mail not found
		selectData, err = client.Select("INBOX", nil).Wait()
		if err != nil {
			return 0, fmt.Errorf("failed to select any mail folder: %w", err)
		}
		selectedMailbox = "INBOX"
		log.Info().Uint32("messages", selectData.NumMessages).Msg("Fallback to INBOX folder")
	}

	log.Info().Str("mailbox", selectedMailbox).Uint32("messages", selectData.NumMessages).Msg("Syncing to local INBOX")

	// Update folder metadata
	uidValidity := int64(selectData.UIDValidity)
	inboxFolder.UIDValidity = &uidValidity
	uidNext := int64(selectData.UIDNext)
	inboxFolder.UIDNext = &uidNext

	// Check for UIDValidity change
	if inboxFolder.UIDValidity != nil {
		oldValidity, _ := s.repo.GetFolderUIDValidity(ctx, inboxFolder.ID)
		if oldValidity > 0 && oldValidity != uidValidity {
			log.Warn().Msg("UIDValidity changed, re-syncing all emails")
			if err := s.repo.DeleteEmailsByFolder(ctx, inboxFolder.ID); err != nil {
				log.Error().Err(err).Msg("Error deleting emails for re-sync")
			}
		}
	}

	if selectData.NumMessages == 0 {
		log.Info().Msg("No messages to sync")
		return 0, s.repo.UpdateFolderCounts(ctx, inboxFolder.ID)
	}

	// Only sync recent messages (last 20) for incremental syncs
	// Full sync only happens on first run or UID validity change
	syncedCount := 0

	// Check if we have any emails - if so, use stored UIDNext for incremental sync
	existingCount, _ := s.repo.GetEmailCountByFolder(ctx, inboxFolder.ID)

	// Determine UID range to fetch
	var startUID imap.UID = 1
	oldUIDNext, _ := s.repo.GetFolderUIDNext(ctx, inboxFolder.ID)
	if existingCount > 0 && oldUIDNext > 0 {
		// Incremental sync: only fetch UIDs >= old UIDNext
		startUID = imap.UID(oldUIDNext)
		if startUID >= imap.UID(uidNext) {
			log.Info().Int64("uidNext", uidNext).Msg("No new messages (UIDNext unchanged)")
			return 0, s.repo.UpdateFolderCounts(ctx, inboxFolder.ID)
		}
		log.Info().Uint32("startUID", uint32(startUID)).Msg("Incremental sync")
	} else {
		log.Info().Msg("Full sync of folder")
	}

	// Close the initial client - we'll get fresh ones for each batch
	s.returnIMAP(account.ID, client)

	for batchStart := startUID; ; {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return syncedCount, ctx.Err()
		default:
		}

		// Get a connection from pool for each batch
		var batchClient *imapclient.Client
		err = withRetry(3, func() error {
			var connectErr error
			batchClient, connectErr = s.connectIMAP(account)
			return connectErr
		})
		if err != nil {
			log.Error().Err(err).Uint32("batchStart", uint32(batchStart)).Msg("Failed to get connection for UID batch")
			break
		}

		// Select the mailbox
		_, err = batchClient.Select(selectedMailbox, nil).Wait()
		if err != nil {
			log.Error().Err(err).Msg("Failed to select mailbox for UID batch")
			s.returnIMAP(account.ID, batchClient)
			break
		}

		// Use UID range: batchStart:* (fetch all remaining UIDs)
		var uidSet imap.UIDSet
		uidSet.AddRange(batchStart, 0) // 0 means * (max UID)

		// Only fetch metadata for fast sync - body is fetched on-demand when viewing
		fetchOptions := &imap.FetchOptions{
			UID:           true,
			Flags:         true,
			Envelope:      true,
			InternalDate:  true,
			BodyStructure: &imap.FetchItemBodyStructure{Extended: false},
		}

		fetchCmd := batchClient.Fetch(uidSet, fetchOptions)
		messages, err := fetchCmd.Collect()

		// Return the connection to pool after fetching
		s.returnIMAP(account.ID, batchClient)

		if err != nil {
			log.Error().Err(err).Uint32("startUID", uint32(batchStart)).Msg("Failed to fetch UIDs")
			break
		}

		if len(messages) == 0 {
			break
		}

		log.Info().Int("count", len(messages)).Uint32("startUID", uint32(batchStart)).Msg("Fetched messages")

		for _, msg := range messages {
			// Parse email and store in INBOX folder
			email := s.parseIMAPMessage(account.ID, inboxFolder.ID, msg)
			if email != nil {
				// Calculate thread ID before saving
				s.calculateThreadID(ctx, email)

				err := s.repo.CreateEmail(ctx, email)
				if err != nil {
					log.Error().Err(err).Uint32("uid", uint32(msg.UID)).Msg("Error saving email")
				} else if email.ID != "" {
					syncedCount++

					// Extract attachment content if there are attachments
					if s.storage != nil {
						attachmentContents := make(map[string][]byte)

						// Clear attachments first - they'll be repopulated with deduplication
						email.Attachments = nil

						// Re-parse with content extraction and deduplication
						for _, section := range msg.BodySection {
							if len(section.Bytes) > 0 {
								s.parseEmailBody(email, section.Bytes, attachmentContents)
							}
						}

						// Save attachments with content to MinIO
						for i, att := range email.Attachments {
							att.EmailID = email.ID

							// Find content by matching size (since we deduplicated by content)
							for _, content := range attachmentContents {
								if int64(len(content)) == att.Size {
									storageKey := fmt.Sprintf("email-attachments/%s/%s/%d_%s", account.ID, email.ID, i, att.Filename)

									err := s.storage.Upload(ctx, storageKey, bytes.NewReader(content), int64(len(content)), att.ContentType)
									if err != nil {
										log.Error().Err(err).Str("filename", att.Filename).Msg("Error uploading attachment")
									} else {
										att.StorageKey = storageKey
									}
									break
								}
							}

							if err := s.repo.CreateAttachment(ctx, &att); err != nil {
								log.Error().Err(err).Str("emailID", email.ID).Msg("Error saving attachment")
							}
							email.Attachments[i] = att
						}
					} else {
						// No storage or no attachments, just save metadata
						for _, att := range email.Attachments {
							att.EmailID = email.ID
							if err := s.repo.CreateAttachment(ctx, &att); err != nil {
								log.Error().Err(err).Str("emailID", email.ID).Msg("Error saving attachment")
							}
						}
					}

					s.ApplyRules(ctx, email)
				}
			} else {
				log.Warn().Uint32("uid", uint32(msg.UID)).Msg("Failed to parse email (no envelope)")
			}
		}

		// We fetched batchStart:* so all UIDs are covered in one pass
		break
	}

	log.Info().Int("count", syncedCount).Msg("Synced new emails to INBOX")

	// Persist the UIDValidity and UIDNext for incremental sync next time
	s.repo.UpdateFolderUIDState(ctx, inboxFolder.ID, *inboxFolder.UIDValidity, *inboxFolder.UIDNext)

	return syncedCount, s.repo.UpdateFolderCounts(ctx, inboxFolder.ID)
}

// syncSentFolder syncs emails from the Sent folder on the server
func (s *EmailService) syncSentFolder(ctx context.Context, account *models.EmailAccount) (int, error) {
	// Get the local Sent folder
	folders, err := s.repo.GetFoldersByAccount(ctx, account.ID)
	if err != nil {
		return 0, err
	}

	var sentFolder *models.EmailFolder
	for _, f := range folders {
		if f.FolderType != nil && *f.FolderType == "sent" {
			sentFolder = &f
			break
		}
	}

	if sentFolder == nil {
		log.Info().Msg("Sent folder not found, skipping")
		return 0, nil
	}

	client, err := s.connectIMAP(account)
	if err != nil {
		return 0, err
	}
	defer s.returnIMAP(account.ID, client)

	// Try different names for Gmail Sent folder
	sentFolderNames := []string{
		"[Gmail]/Sent Mail",
		"[Google Mail]/Sent Mail",
		"Sent",
		"Sent Messages",
		"INBOX.Sent",
	}

	var selectedMailbox string
	var selectData *imap.SelectData

	for _, name := range sentFolderNames {
		selectData, err = client.Select(name, nil).Wait()
		if err == nil {
			selectedMailbox = name
			break
		}
	}

	if selectedMailbox == "" {
		log.Warn().Msg("Could not find Sent folder on server")
		return 0, nil
	}

	totalMessages := selectData.NumMessages
	if totalMessages == 0 {
		log.Info().Msg("No sent messages to sync")
		return 0, nil
	}

	// Check how many sent emails we already have locally
	localSentCount, _ := s.repo.GetEmailCountByFolder(ctx, sentFolder.ID)

	// Use UID-based fetch for reliable incremental sync
	uidValidity := int64(selectData.UIDValidity)
	sentUIDNext := int64(selectData.UIDNext)

	var startUID imap.UID = 1
	oldUIDNext, _ := s.repo.GetFolderUIDNext(ctx, sentFolder.ID)
	if localSentCount > 0 && oldUIDNext > 0 {
		startUID = imap.UID(oldUIDNext)
		if startUID >= imap.UID(sentUIDNext) {
			log.Info().Int64("uidNext", sentUIDNext).Msg("Sent: no new messages (UIDNext unchanged)")
			return 0, nil
		}
		log.Info().Uint32("startUID", uint32(startUID)).Msg("Sent incremental sync")
	} else {
		log.Info().Msg("Sent initial: fetching all UIDs")
	}

	// Update folder UID metadata
	sentFolder.UIDValidity = &uidValidity
	sentFolder.UIDNext = &sentUIDNext

	var uidSet imap.UIDSet
	uidSet.AddRange(startUID, 0) // 0 means * (max UID)

	// Only fetch metadata for fast sync - body is fetched on-demand when viewing
	fetchOptions := &imap.FetchOptions{
		UID:           true,
		Flags:         true,
		Envelope:      true,
		InternalDate:  true,
		BodyStructure: &imap.FetchItemBodyStructure{Extended: false},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	messages, err := fetchCmd.Collect()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch sent messages: %w", err)
	}

	syncedCount := 0
	for _, msg := range messages {
		email := s.parseIMAPMessage(account.ID, sentFolder.ID, msg)
		if email != nil {
			s.calculateThreadID(ctx, email)
			err := s.repo.CreateEmail(ctx, email)
			if err == nil && email.ID != "" {
				syncedCount++
			}
		}
	}

	log.Info().Int("count", syncedCount).Msg("Synced sent emails")
	// Persist UID state so next sync is incremental
	if err := s.repo.UpdateFolderUIDState(ctx, sentFolder.ID, uidValidity, sentUIDNext); err != nil {
		log.Error().Err(err).Str("folder", sentFolder.ID).Msg("Failed to persist sent folder UID state")
	}
	return syncedCount, s.repo.UpdateFolderCounts(ctx, sentFolder.ID)
}

func (s *EmailService) syncFolders(ctx context.Context, client *imapclient.Client, account *models.EmailAccount) error {
	// List all mailboxes
	listCmd := client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	// Only sync system folders (inbox, sent, drafts, trash, spam)
	// Skip custom folders/labels - those are created locally by the user
	systemFolderTypes := map[string]bool{
		"inbox":  true,
		"sent":   true,
		"drafts": true,
		"trash":  true,
		"spam":   true,
	}

	// First pass: create only system folders
	for _, mbox := range mailboxes {
		folderType := s.detectFolderType(mbox.Mailbox, mbox.Attrs)

		// Skip non-system folders - user creates these locally
		if !systemFolderTypes[folderType] {
			continue
		}

		delimiter := ""
		if mbox.Delim != 0 {
			delimiter = string(mbox.Delim)
		}

		folder := &models.EmailFolder{
			AccountID:  account.ID,
			Name:       s.friendlyFolderName(mbox.Mailbox),
			RemoteName: mbox.Mailbox,
			FolderType: &folderType,
			Delimiter:  &delimiter,
		}

		if err := s.repo.UpsertFolder(ctx, folder); err != nil {
			return err
		}
	}

	// No need for second pass to set parent_id since we're only syncing
	// top-level system folders. Local folders handle their own hierarchy.

	return nil
}

func (s *EmailService) detectFolderType(name string, attrs []imap.MailboxAttr) string {
	nameLower := strings.ToLower(name)

	// Check IMAP attributes first
	for _, attr := range attrs {
		switch attr {
		case imap.MailboxAttrSent:
			return "sent"
		case imap.MailboxAttrDrafts:
			return "drafts"
		case imap.MailboxAttrJunk:
			return "spam"
		case imap.MailboxAttrTrash:
			return "trash"
		case imap.MailboxAttrArchive:
			return "archive"
		}
	}

	// Fallback to name matching
	if nameLower == "inbox" {
		return "inbox"
	}
	if strings.Contains(nameLower, "sent") {
		return "sent"
	}
	if strings.Contains(nameLower, "draft") {
		return "drafts"
	}
	if strings.Contains(nameLower, "trash") || strings.Contains(nameLower, "deleted") {
		return "trash"
	}
	if strings.Contains(nameLower, "spam") || strings.Contains(nameLower, "junk") {
		return "spam"
	}
	if strings.Contains(nameLower, "archive") {
		return "archive"
	}

	return "custom"
}

func (s *EmailService) friendlyFolderName(remoteName string) string {
	// Extract last part of folder path
	parts := strings.Split(remoteName, "/")
	name := parts[len(parts)-1]

	// Also handle backslash separator
	parts = strings.Split(name, "\\")
	name = parts[len(parts)-1]

	return name
}

func (s *EmailService) syncFolderEmails(ctx context.Context, client *imapclient.Client, account *models.EmailAccount, folder *models.EmailFolder) error {
	_, err := s.syncFolderEmailsWithCount(ctx, client, account, folder)
	return err
}

func (s *EmailService) syncFolderEmailsWithCount(ctx context.Context, client *imapclient.Client, account *models.EmailAccount, folder *models.EmailFolder) (int, error) {
	// Select the folder
	selectData, err := client.Select(folder.RemoteName, nil).Wait()
	if err != nil {
		return 0, fmt.Errorf("failed to select folder: %w", err)
	}

	log.Info().Str("folder", folder.Name).Uint32("messages", selectData.NumMessages).Msg("Folder sync started")

	// Update folder metadata - UIDValidity is uint32 in go-imap v2
	uidValidity := int64(selectData.UIDValidity)
	folder.UIDValidity = &uidValidity
	uidNext := int64(selectData.UIDNext)
	folder.UIDNext = &uidNext

	// Check for UIDValidity change - if changed, we need to re-sync everything
	if folder.UIDValidity != nil {
		oldValidity, _ := s.repo.GetFolderUIDValidity(ctx, folder.ID)
		if oldValidity > 0 && oldValidity != uidValidity {
			// UIDValidity changed - delete all cached emails for this folder
			log.Warn().Str("folder", folder.Name).Msg("UIDValidity changed, re-syncing all emails")
			if err := s.repo.DeleteEmailsByFolder(ctx, folder.ID); err != nil {
				log.Error().Err(err).Str("folder", folder.Name).Msg("Error deleting emails for re-sync")
			}
		}
	}

	// Get count of emails we have locally for this folder
	localCount, err := s.repo.GetEmailCountByFolder(ctx, folder.ID)
	if err != nil {
		localCount = 0
	}

	log.Info().Str("folder", folder.Name).Int("local", localCount).Uint32("server", selectData.NumMessages).Msg("Folder email counts")

	if selectData.NumMessages == 0 {
		log.Info().Str("folder", folder.Name).Msg("No messages to sync")
		return 0, s.repo.UpdateFolderCounts(ctx, folder.ID)
	}

	// Use UID-based incremental sync for reliability
	syncedCount := 0

	var startUID imap.UID = 1
	oldUIDNext, _ := s.repo.GetFolderUIDNext(ctx, folder.ID)
	if localCount > 0 && oldUIDNext > 0 {
		startUID = imap.UID(oldUIDNext)
		if startUID >= imap.UID(uidNext) {
			log.Info().Str("folder", folder.Name).Int64("uidNext", uidNext).Msg("No new messages (UIDNext unchanged)")
			return 0, s.repo.UpdateFolderCounts(ctx, folder.ID)
		}
		log.Info().Str("folder", folder.Name).Uint32("startUID", uint32(startUID)).Msg("Incremental sync")
	} else {
		log.Info().Str("folder", folder.Name).Msg("Full UID sync")
	}

	var uidSet imap.UIDSet
	uidSet.AddRange(startUID, 0) // 0 means * (max UID)

	fetchOptions := &imap.FetchOptions{
		UID:          true,
		Flags:        true,
		Envelope:     true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{{Peek: true}},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	messages, err := fetchCmd.Collect()
	if err != nil {
		log.Error().Err(err).Str("folder", folder.Name).Msg("Failed to fetch UIDs")
		return 0, err
	}

	log.Info().Str("folder", folder.Name).Int("count", len(messages)).Msg("Fetched messages")

	for _, msg := range messages {
		email := s.parseIMAPMessage(account.ID, folder.ID, msg)
		if email != nil {
			// Calculate thread ID before saving
			s.calculateThreadID(ctx, email)

			err := s.repo.CreateEmail(ctx, email)
			if err != nil {
				log.Error().Err(err).Uint32("uid", uint32(msg.UID)).Str("folder", folder.Name).Msg("Error saving email")
			} else if email.ID != "" {
				// Only count if we actually inserted (ID was set)
				syncedCount++

				// Save attachments if any
				for _, att := range email.Attachments {
					att.EmailID = email.ID
					if err := s.repo.CreateAttachment(ctx, &att); err != nil {
						log.Error().Err(err).Str("emailID", email.ID).Msg("Error saving attachment")
					}
				}

				s.ApplyRules(ctx, email)
			}
		} else {
			log.Warn().Uint32("uid", uint32(msg.UID)).Str("folder", folder.Name).Msg("Failed to parse email (no envelope)")
		}
	}

	log.Info().Str("folder", folder.Name).Int("count", syncedCount).Msg("Synced new emails")

	// Persist the UIDValidity and UIDNext for incremental sync next time
	s.repo.UpdateFolderUIDState(ctx, folder.ID, *folder.UIDValidity, *folder.UIDNext)

	return syncedCount, s.repo.UpdateFolderCounts(ctx, folder.ID)
}

func (s *EmailService) parseIMAPMessage(accountID, folderID string, msg *imapclient.FetchMessageBuffer) *models.Email {
	envelope := msg.Envelope
	if envelope == nil {
		return nil
	}

	email := &models.Email{
		AccountID: accountID,
		FolderID:  folderID,
		MessageID: sanitizeUTF8(envelope.MessageID),
		UID:       int64(msg.UID),
		Subject:   sanitizeUTF8(envelope.Subject),
		Date:      envelope.Date.UTC(),
	}

	// Set ReceivedAt from IMAP InternalDate (server receipt time) - normalize to UTC
	if !msg.InternalDate.IsZero() {
		email.ReceivedAt = msg.InternalDate.UTC()
	} else if !envelope.Date.IsZero() {
		email.ReceivedAt = envelope.Date.UTC()
	} else {
		email.ReceivedAt = time.Now().UTC()
	}

	// Parse From
	if len(envelope.From) > 0 {
		email.FromAddress = sanitizeUTF8(envelope.From[0].Addr())
		email.FromName = sanitizeUTF8(envelope.From[0].Name)
	}

	// Parse To
	for _, addr := range envelope.To {
		email.To = append(email.To, models.EmailAddress{
			Name:    sanitizeUTF8(addr.Name),
			Address: sanitizeUTF8(addr.Addr()),
		})
	}

	// Parse CC
	for _, addr := range envelope.Cc {
		email.CC = append(email.CC, models.EmailAddress{
			Name:    sanitizeUTF8(addr.Name),
			Address: sanitizeUTF8(addr.Addr()),
		})
	}

	// Parse Reply-To
	if len(envelope.ReplyTo) > 0 {
		email.ReplyTo = sanitizeUTF8(envelope.ReplyTo[0].Addr())
	}

	// Parse InReplyTo ([]string in go-imap v2)
	if len(envelope.InReplyTo) > 0 {
		email.InReplyTo = sanitizeUTF8(strings.Join(envelope.InReplyTo, " "))
	}

	// Parse flags
	for _, flag := range msg.Flags {
		switch flag {
		case imap.FlagSeen:
			email.IsRead = true
		case imap.FlagFlagged:
			email.IsStarred = true
		case imap.FlagAnswered:
			email.IsAnswered = true
		case imap.FlagDraft:
			email.IsDraft = true
		}
	}

	// Parse body from BodySection (without attachment content extraction)
	for _, section := range msg.BodySection {
		if len(section.Bytes) > 0 {
			s.parseEmailBody(email, section.Bytes, nil)
		}
	}

	// Detect attachments from BodyStructure (available even in metadata-only sync)
	if msg.BodyStructure != nil && !email.HasAttachments {
		s.detectAttachmentsFromStructure(email, msg.BodyStructure)
	}

	// Generate snippet
	if email.Snippet == "" && email.TextBody != "" {
		email.Snippet = s.generateSnippet(email.TextBody, 150)
	}

	return email
}

// detectAttachmentsFromStructure walks the IMAP BODYSTRUCTURE to detect
// attachments without downloading message content. This is used during
// metadata-only sync to populate HasAttachments and attachment records.
func (s *EmailService) detectAttachmentsFromStructure(email *models.Email, bs imap.BodyStructure) {
	bs.Walk(func(path []int, part imap.BodyStructure) bool {
		sp, ok := part.(*imap.BodyStructureSinglePart)
		if !ok {
			return true // continue into children of multipart
		}

		// Check if this part is an attachment
		isAttachment := false
		var filename string

		// Check disposition
		if disp := sp.Disposition(); disp != nil {
			if strings.EqualFold(disp.Value, "attachment") {
				isAttachment = true
				filename = disp.Params["filename"]
			}
		}

		// Check for filename in params (Content-Type name=)
		if !isAttachment {
			filename = sp.Filename()
			if filename != "" {
				isAttachment = true
			}
		}

		// Skip text/plain and text/html without explicit attachment disposition
		if !isAttachment {
			return true
		}

		email.HasAttachments = true

		if filename == "" {
			filename = s.generateFilename(sp.MediaType())
		}

		att := models.EmailAttachment{
			Filename:    filename,
			ContentType: sp.MediaType(),
			Size:        int64(sp.Size),
			IsInline:    sp.Disposition() != nil && strings.EqualFold(sp.Disposition().Value, "inline"),
		}
		if sp.ID != "" {
			att.ContentID = strings.Trim(sp.ID, "<>")
		}

		email.Attachments = append(email.Attachments, att)
		return true
	})
}

// calculateThreadID determines the thread ID for an email
// Thread ID is set to the root message's Message-ID
// For new threads (no In-Reply-To or References), use own Message-ID
func (s *EmailService) calculateThreadID(ctx context.Context, email *models.Email) {
	// If already has a thread ID, skip
	if email.ThreadID != "" {
		return
	}

	// Strategy:
	// 1. Check References header - the first message-id is typically the thread root
	// 2. Check In-Reply-To - look up that message's thread_id
	// 3. If neither exist, this is a new thread - use own message_id

	// Try References header first (most reliable)
	if email.ReferencesHeader != "" {
		// References contains space or newline separated message IDs
		// The first one is typically the thread root
		refs := strings.Fields(email.ReferencesHeader)
		if len(refs) > 0 {
			rootRef := strings.Trim(refs[0], "<>")
			// Look up if this root message exists in our DB
			existing, err := s.repo.GetEmailByMessageID(ctx, email.AccountID, rootRef)
			if err == nil && existing != nil && existing.ThreadID != "" {
				email.ThreadID = existing.ThreadID
				return
			}
			// Root might not exist yet, use the root reference as thread ID
			email.ThreadID = rootRef
			return
		}
	}

	// Try In-Reply-To
	if email.InReplyTo != "" {
		inReplyTo := strings.Trim(strings.Fields(email.InReplyTo)[0], "<>")
		// Look up the parent email
		parent, err := s.repo.GetEmailByMessageID(ctx, email.AccountID, inReplyTo)
		if err == nil && parent != nil {
			if parent.ThreadID != "" {
				// Use parent's thread ID
				email.ThreadID = parent.ThreadID
			} else {
				// Parent doesn't have thread_id, use parent's message_id
				email.ThreadID = parent.MessageID
			}
			return
		}
		// Parent not found yet, use in_reply_to as thread ID
		// (it references the root or an earlier message in thread)
		email.ThreadID = inReplyTo
		return
	}

	// No threading info - this is a new thread root
	// Use own Message-ID as thread ID
	if email.MessageID != "" {
		email.ThreadID = strings.Trim(email.MessageID, "<>")
	}
}

func (s *EmailService) parseEmailBody(email *models.Email, body []byte, attachmentContents map[string][]byte) {
	msg, err := mail.ReadMessage(strings.NewReader(string(body)))
	if err != nil {
		// Fallback: treat entire body as text
		email.TextBody = sanitizeUTF8(string(body))
		return
	}

	// Extract References header for threading
	references := msg.Header.Get("References")
	if references != "" {
		email.ReferencesHeader = sanitizeUTF8(references)
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	contentTransferEncoding := strings.ToLower(msg.Header.Get("Content-Transfer-Encoding"))
	mediaType, params, _ := mime.ParseMediaType(contentType)
	charset := strings.ToLower(params["charset"])

	if strings.HasPrefix(mediaType, "multipart/") {
		s.parseMultipart(email, msg.Body, params["boundary"], attachmentContents)
	} else if mediaType == "text/html" {
		bodyBytes, _ := io.ReadAll(msg.Body)
		decoded := decodeContent(bodyBytes, contentTransferEncoding, charset)
		email.HTMLBody = sanitizeUTF8(decoded)
	} else {
		bodyBytes, _ := io.ReadAll(msg.Body)
		decoded := decodeContent(bodyBytes, contentTransferEncoding, charset)
		email.TextBody = sanitizeUTF8(decoded)
	}
}

// decodeContent decodes content based on Content-Transfer-Encoding
func decodeContent(data []byte, encoding string, charset string) string {
	var decoded []byte
	var err error

	switch encoding {
	case "quoted-printable":
		reader := quotedprintable.NewReader(bytes.NewReader(data))
		decoded, err = io.ReadAll(reader)
		if err != nil {
			decoded = data // Fallback to original
		}
	case "base64":
		decoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			// Try with padding fixes
			s := strings.ReplaceAll(string(data), "\r\n", "")
			s = strings.ReplaceAll(s, "\n", "")
			s = strings.ReplaceAll(s, " ", "")
			decoded, err = base64.StdEncoding.DecodeString(s)
			if err != nil {
				decoded = data // Fallback to original
			}
		}
	default:
		decoded = data
	}

	// Handle charset conversion if needed (basic UTF-8 handling)
	return string(decoded)
}

func (s *EmailService) parseMultipart(email *models.Email, r io.Reader, boundary string, attachmentContents map[string][]byte) {
	if boundary == "" {
		return
	}

	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}

		contentType := part.Header.Get("Content-Type")
		contentTransferEncoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
		mediaType, params, _ := mime.ParseMediaType(contentType)
		charset := strings.ToLower(params["charset"])
		contentDisposition := part.Header.Get("Content-Disposition")

		if strings.HasPrefix(mediaType, "multipart/") {
			s.parseMultipart(email, part, params["boundary"], attachmentContents)
		} else if mediaType == "text/plain" && email.TextBody == "" && !strings.HasPrefix(contentDisposition, "attachment") {
			bodyBytes, _ := io.ReadAll(part)
			decoded := decodeContent(bodyBytes, contentTransferEncoding, charset)
			email.TextBody = sanitizeUTF8(decoded)
		} else if mediaType == "text/html" && email.HTMLBody == "" && !strings.HasPrefix(contentDisposition, "attachment") {
			bodyBytes, _ := io.ReadAll(part)
			decoded := decodeContent(bodyBytes, contentTransferEncoding, charset)
			email.HTMLBody = sanitizeUTF8(decoded)
		} else if s.isAttachment(part, mediaType, contentDisposition) {
			// Attachment - mark that email has attachments and collect attachment info
			email.HasAttachments = true

			// Read attachment content
			bodyBytes, err := io.ReadAll(part)
			if err != nil {
				part.Close()
				continue
			}

			// Decode the content based on transfer encoding
			var content []byte
			switch contentTransferEncoding {
			case "base64":
				s := strings.ReplaceAll(string(bodyBytes), "\r\n", "")
				s = strings.ReplaceAll(s, "\n", "")
				content, err = base64.StdEncoding.DecodeString(s)
				if err != nil {
					content = bodyBytes
				}
			case "quoted-printable":
				reader := quotedprintable.NewReader(bytes.NewReader(bodyBytes))
				content, _ = io.ReadAll(reader)
			default:
				content = bodyBytes
			}

			// Get Content-ID for inline attachments
			contentID := part.Header.Get("Content-Id")
			if contentID != "" {
				contentID = strings.Trim(contentID, "<>")
			}

			// Determine if inline
			isInline := strings.HasPrefix(contentDisposition, "inline")

			// Get filename - use "noname" if not provided (like Gmail does)
			filename := part.FileName()
			if filename == "" {
				// Try to get from Content-Type name parameter
				if name, ok := params["name"]; ok && name != "" {
					filename = name
				} else {
					// Generate filename based on content type
					filename = s.generateFilename(mediaType)
				}
			}

			// Create a unique key for deduplication (filename + size + content hash prefix)
			dedupeKey := fmt.Sprintf("%s:%d:%x", filename, len(content), content[:min(32, len(content))])

			// Check if we already have this exact attachment
			isDuplicate := false
			if attachmentContents != nil {
				if _, exists := attachmentContents[dedupeKey]; exists {
					isDuplicate = true
				}
			}

			if !isDuplicate {
				attachment := models.EmailAttachment{
					Filename:    filename,
					ContentType: mediaType,
					ContentID:   contentID,
					IsInline:    isInline,
					Size:        int64(len(content)),
				}

				email.Attachments = append(email.Attachments, attachment)

				// Store content by unique key for later upload to MinIO
				if attachmentContents != nil {
					attachmentContents[dedupeKey] = content
				}
			}
		}
		part.Close()
	}
}

// isAttachment determines if a MIME part should be treated as an attachment
func (s *EmailService) isAttachment(part *multipart.Part, mediaType, contentDisposition string) bool {
	// Explicit attachment disposition
	if strings.HasPrefix(contentDisposition, "attachment") {
		return true
	}

	// Has a filename
	if part.FileName() != "" {
		return true
	}

	// Inline with Content-ID (embedded image) - treat as attachment
	if strings.HasPrefix(contentDisposition, "inline") && part.Header.Get("Content-Id") != "" {
		return true
	}

	// Common attachment MIME types without filename
	attachmentTypes := []string{
		"application/pdf",
		"application/zip",
		"application/x-zip-compressed",
		"application/msword",
		"application/vnd.openxmlformats",
		"application/vnd.ms-excel",
		"application/octet-stream",
		"image/",
		"audio/",
		"video/",
	}

	for _, t := range attachmentTypes {
		if strings.HasPrefix(mediaType, t) {
			return true
		}
	}

	return false
}

// generateFilename creates a filename for attachments without one
func (s *EmailService) generateFilename(mediaType string) string {
	// Map common MIME types to extensions
	extensions := map[string]string{
		"application/pdf":              "pdf",
		"application/zip":              "zip",
		"application/x-zip-compressed": "zip",
		"application/msword":           "doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
		"application/vnd.ms-excel": "xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
		"application/vnd.ms-powerpoint":                                             "ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
		"application/octet-stream":                                                  "bin",
		"image/jpeg":                                                                "jpg",
		"image/png":                                                                 "png",
		"image/gif":                                                                 "gif",
		"image/webp":                                                                "webp",
		"text/plain":                                                                "txt",
		"text/csv":                                                                  "csv",
	}

	if ext, ok := extensions[mediaType]; ok {
		return "noname." + ext
	}

	// Try to extract extension from media type
	parts := strings.Split(mediaType, "/")
	if len(parts) == 2 {
		return "noname." + parts[1]
	}

	return "noname"
}

func (s *EmailService) generateSnippet(text string, maxLen int) string {
	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// sanitizeUTF8 ensures a string is valid UTF-8 by removing invalid bytes
func sanitizeUTF8(s string) string {
	// If already valid, return as-is (fast path)
	if utf8.ValidString(s) {
		return s
	}

	// Build a new string with only valid UTF-8 characters
	var result strings.Builder
	result.Grow(len(s))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError {
			// Skip invalid bytes entirely (don't use replacement char)
			if size == 0 {
				break
			}
			i += size
			continue
		}
		// Only write valid runes
		result.WriteRune(r)
		i += size
	}

	return result.String()
}

// ============ SMTP Operations ============

func (s *EmailService) testSMTPConnection(account *models.EmailAccount) error {
	addr := fmt.Sprintf("%s:%d", account.SMTPHost, account.SMTPPort)

	var client *smtp.Client
	var err error

	if account.SMTPUseTLS && account.SMTPPort == 465 {
		// Implicit TLS (SMTPS)
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: account.SMTPHost})
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		client, err = smtp.NewClient(conn, account.SMTPHost)
		if err != nil {
			return fmt.Errorf("SMTP client failed: %w", err)
		}
	} else {
		// Plain or STARTTLS
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}

		if account.SMTPUseTLS {
			if err := client.StartTLS(&tls.Config{ServerName: account.SMTPHost}); err != nil {
				client.Close()
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", account.SMTPUsername, account.SMTPPassword, account.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

func (s *EmailService) SendEmail(ctx context.Context, accountID string, compose *models.ComposeEmail) error {
	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	// Decrypt passwords for SMTP authentication
	if err := s.decryptAccountPasswords(account); err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Build email message
	msg := s.buildEmailMessage(account, compose)

	// Get all recipients
	var recipients []string
	for _, addr := range compose.To {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range compose.CC {
		recipients = append(recipients, addr.Address)
	}
	for _, addr := range compose.BCC {
		recipients = append(recipients, addr.Address)
	}

	// Send via SMTP
	addr := fmt.Sprintf("%s:%d", account.SMTPHost, account.SMTPPort)
	auth := smtp.PlainAuth("", account.SMTPUsername, account.SMTPPassword, account.SMTPHost)

	if account.SMTPUseTLS && account.SMTPPort == 465 {
		// Implicit TLS
		return s.sendWithTLS(addr, account, auth, msg, recipients)
	}

	// STARTTLS or plain
	if account.SMTPUseTLS {
		return s.sendWithSTARTTLS(addr, account, auth, msg, recipients)
	}

	return smtp.SendMail(addr, auth, account.EmailAddress, recipients, []byte(msg))
}

func (s *EmailService) buildEmailMessage(account *models.EmailAccount, compose *models.ComposeEmail) string {
	var sb strings.Builder

	// Headers
	sb.WriteString(fmt.Sprintf("From: %s <%s>\r\n", account.Name, account.EmailAddress))

	var toAddrs []string
	for _, addr := range compose.To {
		if addr.Name != "" {
			toAddrs = append(toAddrs, fmt.Sprintf("%s <%s>", addr.Name, addr.Address))
		} else {
			toAddrs = append(toAddrs, addr.Address)
		}
	}
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toAddrs, ", ")))

	if len(compose.CC) > 0 {
		var ccAddrs []string
		for _, addr := range compose.CC {
			if addr.Name != "" {
				ccAddrs = append(ccAddrs, fmt.Sprintf("%s <%s>", addr.Name, addr.Address))
			} else {
				ccAddrs = append(ccAddrs, addr.Address)
			}
		}
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccAddrs, ", ")))
	}

	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", compose.Subject))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	sb.WriteString(fmt.Sprintf("Message-ID: <%s@%s>\r\n", generateID(), account.SMTPHost))
	sb.WriteString("MIME-Version: 1.0\r\n")

	if len(compose.FileAttachments) > 0 {
		// Multipart mixed message with attachments
		boundary := fmt.Sprintf("----=_Part_%d", time.Now().UnixNano())
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		sb.WriteString("\r\n")

		// Body part
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		if compose.IsHTML {
			sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		sb.WriteString("\r\n")
		// Encode body with quoted-printable
		var qpBuf bytes.Buffer
		qpWriter := quotedprintable.NewWriter(&qpBuf)
		qpWriter.Write([]byte(compose.Body))
		qpWriter.Close()
		sb.WriteString(qpBuf.String())
		sb.WriteString("\r\n")

		// Attachment parts
		for _, att := range compose.FileAttachments {
			sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			sb.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename))
			sb.WriteString("Content-Transfer-Encoding: base64\r\n")
			sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Filename))
			sb.WriteString("\r\n")

			// Base64 encode with 76-char line wrapping
			encoded := base64.StdEncoding.EncodeToString(att.Data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				sb.WriteString(encoded[i:end])
				sb.WriteString("\r\n")
			}
		}

		// Closing boundary
		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// Simple message without attachments
		if compose.IsHTML {
			sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		sb.WriteString("\r\n")
		var qpBuf bytes.Buffer
		qpWriter := quotedprintable.NewWriter(&qpBuf)
		qpWriter.Write([]byte(compose.Body))
		qpWriter.Close()
		sb.WriteString(qpBuf.String())
	}

	return sb.String()
}

func (s *EmailService) sendWithTLS(addr string, account *models.EmailAccount, auth smtp.Auth, msg string, recipients []string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: account.SMTPHost})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, account.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(account.EmailAddress); err != nil {
		return err
	}

	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}

	return w.Close()
}

func (s *EmailService) sendWithSTARTTLS(addr string, account *models.EmailAccount, auth smtp.Auth, msg string, recipients []string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: account.SMTPHost}); err != nil {
		return err
	}

	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(account.EmailAddress); err != nil {
		return err
	}

	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}

	return w.Close()
}

func generateID() string {
	return fmt.Sprintf("%d.%d", time.Now().UnixNano(), time.Now().Unix())
}

// ============ Folder & Email Operations ============

func (s *EmailService) GetFolders(ctx context.Context, accountID string) ([]models.EmailFolder, error) {
	return s.repo.GetFoldersByAccount(ctx, accountID)
}

// GetFoldersAsTree returns folders organized as a tree structure with children populated
func (s *EmailService) GetFoldersAsTree(ctx context.Context, accountID string) ([]*models.EmailFolder, error) {
	folders, err := s.repo.GetFoldersByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookups
	folderMap := make(map[string]*models.EmailFolder)
	for i := range folders {
		f := &folders[i]
		f.Children = []*models.EmailFolder{} // Initialize children slice
		folderMap[f.ID] = f
	}

	// Build tree structure
	var roots []*models.EmailFolder
	for i := range folders {
		f := folderMap[folders[i].ID]
		if f.ParentID != nil && *f.ParentID != "" {
			if parent, ok := folderMap[*f.ParentID]; ok {
				parent.Children = append(parent.Children, f)
				continue
			}
		}
		// No parent or parent not found - this is a root folder
		roots = append(roots, f)
	}

	return roots, nil
}

func (s *EmailService) GetFolder(ctx context.Context, folderID string) (*models.EmailFolder, error) {
	return s.repo.GetFolderByID(ctx, folderID)
}

func (s *EmailService) CreateFolder(ctx context.Context, folder *models.EmailFolder) error {
	// Set folder type to custom for user-created folders
	folderType := "custom"
	folder.FolderType = &folderType

	// Local folders don't have a remote_name - they only exist locally
	// Use a local: prefix to make this clear and avoid conflicts
	localPrefix := "local:"
	if folder.ParentID != nil && *folder.ParentID != "" {
		parent, err := s.repo.GetFolderByID(ctx, *folder.ParentID)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		folder.RemoteName = localPrefix + parent.Name + "/" + folder.Name
	} else {
		folder.RemoteName = localPrefix + folder.Name
	}

	delimiter := "/"
	folder.Delimiter = &delimiter

	return s.repo.CreateFolder(ctx, folder)
}

func (s *EmailService) UpdateFolder(ctx context.Context, folder *models.EmailFolder) error {
	existing, err := s.repo.GetFolderByID(ctx, folder.ID)
	if err != nil {
		return err
	}

	// Don't allow modifying system folders
	if existing.FolderType != nil && *existing.FolderType != "custom" {
		return fmt.Errorf("cannot modify system folder")
	}

	// Update remote name if name or parent changed
	if folder.Name != existing.Name || (folder.ParentID != nil && existing.ParentID != nil && *folder.ParentID != *existing.ParentID) {
		if folder.ParentID != nil && *folder.ParentID != "" {
			parent, err := s.repo.GetFolderByID(ctx, *folder.ParentID)
			if err != nil {
				return fmt.Errorf("parent folder not found: %w", err)
			}
			delimiter := "/"
			if parent.Delimiter != nil && *parent.Delimiter != "" {
				delimiter = *parent.Delimiter
			}
			folder.RemoteName = parent.RemoteName + delimiter + folder.Name
		} else {
			folder.RemoteName = folder.Name
		}
	}

	return s.repo.UpdateFolder(ctx, folder)
}

func (s *EmailService) DeleteFolder(ctx context.Context, folderID string) error {
	folder, err := s.repo.GetFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Don't allow deleting system folders
	if folder.FolderType != nil && *folder.FolderType != "custom" {
		return fmt.Errorf("cannot delete system folder")
	}

	// Delete emails in the folder first
	if err := s.repo.DeleteEmailsByFolder(ctx, folderID); err != nil {
		return err
	}

	return s.repo.DeleteFolder(ctx, folderID)
}

func (s *EmailService) ReorderFolders(ctx context.Context, folderOrders []struct {
	ID        string
	SortOrder int
}) error {
	for _, fo := range folderOrders {
		if err := s.repo.UpdateFolderSortOrder(ctx, fo.ID, fo.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func (s *EmailService) MoveFolder(ctx context.Context, folderID string, newParentID *string) error {
	folder, err := s.repo.GetFolderByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Don't allow moving system folders
	if folder.FolderType != nil && *folder.FolderType != "custom" {
		return fmt.Errorf("cannot move system folder")
	}

	// Check for circular reference - can't move a folder into one of its descendants
	if newParentID != nil && *newParentID != "" {
		if err := s.checkCircularReference(ctx, folderID, *newParentID); err != nil {
			return err
		}
	}

	folder.ParentID = newParentID

	// Update remote name
	if newParentID != nil && *newParentID != "" {
		parent, err := s.repo.GetFolderByID(ctx, *newParentID)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		delimiter := "/"
		if parent.Delimiter != nil && *parent.Delimiter != "" {
			delimiter = *parent.Delimiter
		}
		folder.RemoteName = parent.RemoteName + delimiter + folder.Name
	} else {
		folder.RemoteName = folder.Name
	}

	return s.repo.UpdateFolder(ctx, folder)
}

// ReorderFolderRelative moves a folder before or after another folder (reordering)
func (s *EmailService) ReorderFolderRelative(ctx context.Context, folderID, targetID, position string) error {
	folder, err := s.repo.GetFolderByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}

	// Don't allow reordering system folders
	if folder.FolderType != nil && *folder.FolderType != "custom" {
		return fmt.Errorf("cannot reorder system folder")
	}

	target, err := s.repo.GetFolderByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("target folder not found: %w", err)
	}

	// The folder will be placed in the same parent as the target
	folder.ParentID = target.ParentID

	// Update remote name based on new parent
	if folder.ParentID != nil && *folder.ParentID != "" {
		parent, err := s.repo.GetFolderByID(ctx, *folder.ParentID)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		delimiter := "/"
		if parent.Delimiter != nil && *parent.Delimiter != "" {
			delimiter = *parent.Delimiter
		}
		folder.RemoteName = parent.RemoteName + delimiter + folder.Name
	} else {
		folder.RemoteName = folder.Name
	}

	// Get target's sort order
	targetSortOrder := target.SortOrder

	// Calculate new sort order
	var newSortOrder int
	if position == "before" {
		newSortOrder = targetSortOrder
		// Increment all folders at or after target position
		if err := s.repo.IncrementFolderSortOrders(ctx, folder.AccountID, folder.ParentID, newSortOrder); err != nil {
			return err
		}
	} else { // "after"
		newSortOrder = targetSortOrder + 1
		// Increment all folders after target position
		if err := s.repo.IncrementFolderSortOrders(ctx, folder.AccountID, folder.ParentID, newSortOrder); err != nil {
			return err
		}
	}

	folder.SortOrder = newSortOrder

	return s.repo.UpdateFolder(ctx, folder)
}

// checkCircularReference verifies that targetID is not a descendant of folderID
func (s *EmailService) checkCircularReference(ctx context.Context, folderID, targetID string) error {
	// If target is the same as folder, it's circular
	if folderID == targetID {
		return fmt.Errorf("cannot move folder into itself")
	}

	// Walk up the parent chain from targetID to see if we hit folderID
	currentID := targetID
	visited := make(map[string]bool)

	for currentID != "" {
		if visited[currentID] {
			break // Already visited, prevent infinite loop
		}
		visited[currentID] = true

		if currentID == folderID {
			return fmt.Errorf("cannot move folder into one of its subfolders")
		}

		folder, err := s.repo.GetFolderByID(ctx, currentID)
		if err != nil {
			break
		}

		if folder.ParentID == nil {
			break
		}
		currentID = *folder.ParentID
	}

	return nil
}

func (s *EmailService) GetEmails(ctx context.Context, folderID string, page, pageSize int) ([]models.EmailListItem, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetEmailsByFolder(ctx, folderID, pageSize, offset)
}

// GetThreads returns conversation threads for a folder
func (s *EmailService) GetThreads(ctx context.Context, folderID string, page, pageSize int) ([]models.EmailThread, int, error) {
	offset := (page - 1) * pageSize
	threads, err := s.repo.GetThreadsForFolder(ctx, folderID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetThreadCountForFolder(ctx, folderID)
	if err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

// GetThreadEmails returns all emails in a specific thread
func (s *EmailService) GetThreadEmails(ctx context.Context, threadID string) ([]models.EmailListItem, error) {
	return s.repo.GetEmailsByThread(ctx, threadID)
}

// GetThreadConversation returns all full emails in a thread with bodies fetched on-demand
func (s *EmailService) GetThreadConversation(ctx context.Context, threadID string) ([]models.Email, error) {
	emails, err := s.repo.GetFullEmailsByThread(ctx, threadID)
	if err != nil {
		return nil, err
	}

	// Batch fetch bodies for emails that don't have them yet
	if err := s.batchFetchBodies(ctx, emails); err != nil {
		log.Error().Err(err).Str("threadID", threadID).Msg("Error in batch body fetch")
		// Non-fatal - some emails may still have empty bodies
	}

	// Batch load all attachments in a single query
	emailIDs := make([]string, len(emails))
	for i, e := range emails {
		emailIDs[i] = e.ID
	}
	attachmentMap, err := s.repo.GetAttachmentsByEmailIDs(ctx, emailIDs)
	if err == nil {
		for i := range emails {
			if atts, ok := attachmentMap[emails[i].ID]; ok {
				emails[i].Attachments = atts
			}
		}
	}

	return emails, nil
}

// batchFetchEntry represents an email that needs its body fetched from IMAP
type batchFetchEntry struct {
	index    int
	folderID string
	uid      int64
}

// batchFetchBodies fetches email bodies from IMAP in bulk, grouped by folder.
// Uses a single IMAP connection and one FETCH command per folder with all UIDs.
func (s *EmailService) batchFetchBodies(ctx context.Context, emails []models.Email) error {
	// Identify which emails need body fetching
	var toFetch []batchFetchEntry
	for i, e := range emails {
		if e.TextBody == "" && e.HTMLBody == "" && e.UID > 0 {
			toFetch = append(toFetch, batchFetchEntry{index: i, folderID: e.FolderID, uid: e.UID})
		}
	}
	if len(toFetch) == 0 {
		return nil
	}

	// All emails in a thread belong to the same account
	account, err := s.repo.GetAccountByID(ctx, emails[0].AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	s.decryptAccountPasswords(account)

	// Resolve each unique folder to its IMAP remote name
	folderCache := make(map[string]*models.EmailFolder)
	for _, f := range toFetch {
		if _, ok := folderCache[f.folderID]; !ok {
			folder, err := s.repo.GetFolderByID(ctx, f.folderID)
			if err != nil {
				return fmt.Errorf("failed to get folder %s: %w", f.folderID, err)
			}
			folderCache[f.folderID] = folder
		}
	}

	// Group emails by effective IMAP folder name
	type folderGroup struct {
		remoteName string
		entries    []batchFetchEntry
	}
	groups := make(map[string]*folderGroup)
	for _, f := range toFetch {
		folder := folderCache[f.folderID]
		remote := s.resolveRemoteName(folder)
		if g, ok := groups[remote]; ok {
			g.entries = append(g.entries, f)
		} else {
			groups[remote] = &folderGroup{remoteName: remote, entries: []batchFetchEntry{f}}
		}
	}

	// Get a single IMAP connection
	var client *imapclient.Client
	err = withRetry(3, func() error {
		var connectErr error
		client, connectErr = s.connectIMAP(account)
		return connectErr
	})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer s.returnIMAP(account.ID, client)

	// Process each folder group: one SELECT + one FETCH with all UIDs
	for _, group := range groups {
		if err := s.fetchBodiesFromFolder(ctx, client, account, emails, group.entries, group.remoteName); err != nil {
			log.Error().Err(err).Str("folder", group.remoteName).Msg("Error batch-fetching from folder")
			// Continue with other folders
		}
	}

	return nil
}

// resolveRemoteName determines the effective IMAP folder to SELECT for a given local folder
func (s *EmailService) resolveRemoteName(folder *models.EmailFolder) string {
	if folder.FolderType != nil && *folder.FolderType == "inbox" {
		return "[Gmail]/All Mail"
	}
	if folder.RemoteName != "" {
		return folder.RemoteName
	}
	if folder.FolderType != nil {
		switch *folder.FolderType {
		case "sent":
			return "[Gmail]/Sent Mail"
		case "drafts":
			return "[Gmail]/Drafts"
		case "trash":
			return "[Gmail]/Trash"
		}
	}
	return "INBOX"
}

// fetchBodiesFromFolder SELECTs a folder and FETCHes all UIDs in one command
func (s *EmailService) fetchBodiesFromFolder(ctx context.Context, client *imapclient.Client, account *models.EmailAccount, emails []models.Email, entries []batchFetchEntry, remoteName string) error {
	// SELECT the folder (try alternatives for All Mail)
	selected := false
	if remoteName == "[Gmail]/All Mail" {
		for _, name := range []string{"[Gmail]/All Mail", "[Google Mail]/All Mail", "All Mail"} {
			if _, selErr := client.Select(name, nil).Wait(); selErr == nil {
				selected = true
				break
			}
		}
		if !selected {
			if _, selErr := client.Select("INBOX", nil).Wait(); selErr != nil {
				return fmt.Errorf("failed to select folder: %w", selErr)
			}
		}
	} else {
		if _, err := client.Select(remoteName, nil).Wait(); err != nil {
			// Try All Mail as fallback
			if _, err := client.Select("[Gmail]/All Mail", nil).Wait(); err != nil {
				return fmt.Errorf("failed to select folder %s: %w", remoteName, err)
			}
		}
	}

	// Build a single UID set with all UIDs
	var uidSet imap.UIDSet
	uidMap := make(map[imap.UID]int) // UID -> index in emails slice
	for _, entry := range entries {
		uid := imap.UID(entry.uid)
		uidSet.AddNum(uid)
		uidMap[uid] = entry.index
	}

	fetchOptions := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	messages, err := fetchCmd.Collect()
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Process each fetched message
	for _, msg := range messages {
		idx, ok := uidMap[msg.UID]
		if !ok {
			continue
		}
		email := &emails[idx]
		attachmentContents := make(map[string][]byte)

		for _, section := range msg.BodySection {
			if len(section.Bytes) > 0 {
				s.parseEmailBody(email, section.Bytes, attachmentContents)
			}
		}

		if email.Snippet == "" && email.TextBody != "" {
			email.Snippet = s.generateSnippet(email.TextBody, 150)
		}

		// Save attachments to DB and MinIO
		if len(email.Attachments) > 0 {
			for i, att := range email.Attachments {
				att.EmailID = email.ID
				if s.storage != nil {
					for _, content := range attachmentContents {
						if int64(len(content)) == att.Size {
							storageKey := fmt.Sprintf("email-attachments/%s/%s/%d_%s", account.ID, email.ID, i, att.Filename)
							if uploadErr := s.storage.Upload(ctx, storageKey, bytes.NewReader(content), int64(len(content)), att.ContentType); uploadErr == nil {
								att.StorageKey = storageKey
							}
							break
						}
					}
				}
				if err := s.repo.CreateAttachment(ctx, &att); err != nil {
					log.Error().Err(err).Str("emailID", email.ID).Str("filename", att.Filename).Msg("Error saving attachment during batch fetch")
				}
				email.Attachments[i] = att
			}
		}

		// Update DB with body
		if updateErr := s.repo.UpdateEmailBody(ctx, email.ID, email.TextBody, email.HTMLBody, email.Snippet); updateErr != nil {
			log.Error().Err(updateErr).Str("emailID", email.ID).Msg("Error updating email body during batch fetch")
		}
	}

	return nil
}

// ReindexThreads updates thread IDs for existing emails in an account
func (s *EmailService) ReindexThreads(ctx context.Context, accountID string) error {
	return s.repo.UpdateThreadIDsForExisting(ctx, accountID)
}

func (s *EmailService) GetEmail(ctx context.Context, emailID string) (*models.Email, error) {
	email, err := s.repo.GetEmailByID(ctx, emailID)
	if err != nil {
		return nil, err
	}

	// Fetch body on-demand if not already synced
	if email.TextBody == "" && email.HTMLBody == "" && email.UID > 0 {
		if err := s.fetchEmailBody(ctx, email); err != nil {
			log.Error().Err(err).Str("emailID", emailID).Msg("Error fetching body for email")
			// Continue with empty body rather than failing
		}
	}

	// Get attachments
	attachments, err := s.repo.GetAttachmentsByEmail(ctx, emailID)
	if err == nil {
		email.Attachments = attachments
	}

	// Get labels
	labels, err := s.repo.GetLabelsByEmail(ctx, emailID)
	if err == nil {
		email.Labels = labels
	}

	return email, nil
}

// fetchEmailBody fetches the full body of an email from IMAP and updates the database
func (s *EmailService) fetchEmailBody(ctx context.Context, email *models.Email) error {
	// Get account for IMAP connection
	account, err := s.repo.GetAccountByID(ctx, email.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	s.decryptAccountPasswords(account)

	// Get folder for remote name
	folder, err := s.repo.GetFolderByID(ctx, email.FolderID)
	if err != nil {
		return fmt.Errorf("failed to get folder: %w", err)
	}

	var client *imapclient.Client
	err = withRetry(3, func() error {
		var connectErr error
		client, connectErr = s.connectIMAP(account)
		return connectErr
	})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer s.returnIMAP(account.ID, client)

	// Determine remote folder name
	// For inbox-type folders, emails were synced from [Gmail]/All Mail
	// so we must fetch from there (UIDs match All Mail, not INBOX)
	remoteName := folder.RemoteName
	if folder.FolderType != nil && *folder.FolderType == "inbox" {
		// Try All Mail first since inbox UIDs come from there
		allMailNames := []string{
			"[Gmail]/All Mail",
			"[Google Mail]/All Mail",
			"All Mail",
		}
		selected := false
		for _, name := range allMailNames {
			if _, selErr := client.Select(name, nil).Wait(); selErr == nil {
				selected = true
				break
			}
		}
		if !selected {
			// Fallback to INBOX
			if _, selErr := client.Select("INBOX", nil).Wait(); selErr != nil {
				return fmt.Errorf("failed to select folder: %w", selErr)
			}
		}
	} else {
		if remoteName == "" {
			// Map local folder types to Gmail folders
			if folder.FolderType != nil {
				switch *folder.FolderType {
				case "sent":
					remoteName = "[Gmail]/Sent Mail"
				case "drafts":
					remoteName = "[Gmail]/Drafts"
				case "trash":
					remoteName = "[Gmail]/Trash"
				default:
					remoteName = "INBOX"
				}
			}
		}

		_, err = client.Select(remoteName, nil).Wait()
		if err != nil {
			// Try All Mail as fallback
			_, err = client.Select("[Gmail]/All Mail", nil).Wait()
			if err != nil {
				return fmt.Errorf("failed to select folder: %w", err)
			}
		}
	}

	// Fetch body by UID
	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(email.UID))

	fetchOptions := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	messages, err := fetchCmd.Collect()
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("email not found on server")
	}

	msg := messages[0]
	attachmentContents := make(map[string][]byte)
	for _, section := range msg.BodySection {
		if len(section.Bytes) > 0 {
			s.parseEmailBody(email, section.Bytes, attachmentContents)
		}
	}

	// Generate snippet if missing
	if email.Snippet == "" && email.TextBody != "" {
		email.Snippet = s.generateSnippet(email.TextBody, 150)
	}

	// Save any newly discovered attachments to DB and MinIO
	if len(email.Attachments) > 0 {
		for i, att := range email.Attachments {
			att.EmailID = email.ID

			// Try to upload content to MinIO
			if s.storage != nil {
				for _, content := range attachmentContents {
					if int64(len(content)) == att.Size {
						storageKey := fmt.Sprintf("email-attachments/%s/%s/%d_%s", account.ID, email.ID, i, att.Filename)
						if uploadErr := s.storage.Upload(ctx, storageKey, bytes.NewReader(content), int64(len(content)), att.ContentType); uploadErr == nil {
							att.StorageKey = storageKey
						}
						break
					}
				}
			}

			if err := s.repo.CreateAttachment(ctx, &att); err != nil {
				log.Error().Err(err).Str("emailID", email.ID).Str("filename", att.Filename).Msg("Error saving attachment during body fetch")
			}
			email.Attachments[i] = att
		}
	}

	// Update email in database with body
	return s.repo.UpdateEmailBody(ctx, email.ID, email.TextBody, email.HTMLBody, email.Snippet)
}

func (s *EmailService) MarkAsRead(ctx context.Context, emailID string, isRead bool) error {
	return s.repo.MarkAsRead(ctx, emailID, isRead)
}

func (s *EmailService) MarkFolderAsRead(ctx context.Context, folderID string) (int64, error) {
	count, err := s.repo.MarkFolderAsRead(ctx, folderID)
	if err != nil {
		return 0, err
	}
	// Update folder counts after marking as read
	s.repo.UpdateFolderCounts(ctx, folderID)
	return count, nil
}

func (s *EmailService) MarkAsStarred(ctx context.Context, emailID string, isStarred bool) error {
	return s.repo.MarkAsStarred(ctx, emailID, isStarred)
}

func (s *EmailService) MoveEmail(ctx context.Context, emailID, folderID string) error {
	return s.repo.MoveEmail(ctx, emailID, folderID)
}

func (s *EmailService) DeleteEmail(ctx context.Context, emailID string) error {
	return s.repo.DeleteEmail(ctx, emailID)
}

func (s *EmailService) SearchEmails(ctx context.Context, accountID, query string) ([]models.EmailListItem, error) {
	return s.repo.SearchEmails(ctx, accountID, query, 50)
}

// AdvancedSearchEmails supports Gmail-style search operators
// from:, to:, subject:, has:attachment, before:, after:, label:, is:starred, is:unread
func (s *EmailService) AdvancedSearchEmails(ctx context.Context, accountID, query string) ([]models.EmailListItem, error) {
	return s.repo.AdvancedSearchEmails(ctx, accountID, query, 50)
}

// ============ Batch Operations ============

func (s *EmailService) BatchMarkAsRead(ctx context.Context, emailIDs []string, isRead bool) (int64, error) {
	return s.repo.BatchMarkAsRead(ctx, emailIDs, isRead)
}

func (s *EmailService) BatchMarkAsStarred(ctx context.Context, emailIDs []string, isStarred bool) (int64, error) {
	return s.repo.BatchMarkAsStarred(ctx, emailIDs, isStarred)
}

func (s *EmailService) BatchMoveEmails(ctx context.Context, emailIDs []string, targetFolderID string) (int64, error) {
	return s.repo.BatchMoveEmails(ctx, emailIDs, targetFolderID)
}

func (s *EmailService) BatchDeleteEmails(ctx context.Context, emailIDs []string) (int64, error) {
	return s.repo.BatchDeleteEmails(ctx, emailIDs)
}

func (s *EmailService) BatchAssignLabel(ctx context.Context, emailIDs []string, labelID string) (int64, error) {
	return s.repo.BatchAssignLabel(ctx, emailIDs, labelID)
}

// ============ Draft Save/Load ============

func (s *EmailService) SaveDraft(ctx context.Context, draft *models.EmailDraft) error {
	return s.repo.SaveDraft(ctx, draft)
}

func (s *EmailService) GetDraft(ctx context.Context, draftID string) (*models.EmailDraft, error) {
	return s.repo.GetDraftByID(ctx, draftID)
}

func (s *EmailService) GetDrafts(ctx context.Context, accountID string) ([]models.EmailDraft, error) {
	return s.repo.GetDraftsByAccount(ctx, accountID)
}

func (s *EmailService) DeleteDraft(ctx context.Context, draftID string) error {
	return s.repo.DeleteDraft(ctx, draftID)
}

// ============ Undo Send ============

// QueueSend queues an email for delayed sending (undo-send feature)
func (s *EmailService) QueueSend(ctx context.Context, accountID string, compose *models.ComposeEmail, delaySecs int) (string, error) {
	id := generateID()
	pending := &models.PendingSend{
		ID:        id,
		AccountID: accountID,
		Compose:   *compose,
		SendAt:    time.Now().Add(time.Duration(delaySecs) * time.Second),
		Cancelled: false,
	}

	s.pendingSendsLock.Lock()
	s.pendingSends[id] = pending
	s.pendingSendsLock.Unlock()

	// Schedule actual send
	go func() {
		timer := time.NewTimer(time.Duration(delaySecs) * time.Second)
		defer timer.Stop()

		<-timer.C

		s.pendingSendsLock.Lock()
		p, exists := s.pendingSends[id]
		if !exists || p.Cancelled {
			// Already cancelled or removed — clean up and bail
			delete(s.pendingSends, id)
			s.pendingSendsLock.Unlock()
			log.Info().Str("id", id).Msg("Send cancelled by user")
			return
		}
		s.pendingSendsLock.Unlock()

		// Actually send the email (outside the lock)
		if err := s.SendEmail(context.Background(), accountID, &p.Compose); err != nil {
			log.Error().Err(err).Str("id", id).Msg("Failed to send queued email")
		} else {
			log.Info().Str("id", id).Msg("Queued email sent successfully")
		}

		// Clean up after send completes
		s.pendingSendsLock.Lock()
		delete(s.pendingSends, id)
		s.pendingSendsLock.Unlock()
	}()

	return id, nil
}

// CancelSend cancels a queued email
func (s *EmailService) CancelSend(ctx context.Context, sendID string) error {
	s.pendingSendsLock.Lock()
	defer s.pendingSendsLock.Unlock()

	pending, exists := s.pendingSends[sendID]
	if !exists {
		return fmt.Errorf("pending send not found or already sent")
	}

	pending.Cancelled = true
	delete(s.pendingSends, sendID)
	return nil
}

// ============ Attachments ============

func (s *EmailService) GetAttachment(ctx context.Context, attachmentID string) (*models.EmailAttachment, error) {
	return s.repo.GetAttachmentByID(ctx, attachmentID)
}

func (s *EmailService) DownloadAttachment(ctx context.Context, attachmentID string) (*models.EmailAttachment, []byte, error) {
	// Get the attachment metadata
	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found: %w", err)
	}

	// If attachment is stored in MinIO, serve from there (fast path)
	if attachment.StorageKey != "" && s.storage != nil {
		reader, err := s.storage.Download(ctx, attachment.StorageKey)
		if err == nil {
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err == nil {
				return attachment, data, nil
			}
		}
		// Fall through to IMAP fetch if MinIO fails
	}

	// Fallback: Fetch from IMAP (slow path - for attachments not yet cached)
	email, err := s.repo.GetEmailByID(ctx, attachment.EmailID)
	if err != nil {
		return nil, nil, fmt.Errorf("email not found: %w", err)
	}

	folder, err := s.repo.GetFolderByID(ctx, email.FolderID)
	if err != nil {
		return nil, nil, fmt.Errorf("folder not found: %w", err)
	}

	account, err := s.repo.GetAccountByID(ctx, folder.AccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("account not found: %w", err)
	}
	s.decryptAccountPasswords(account)

	var client *imapclient.Client
	err = withRetry(3, func() error {
		var connectErr error
		client, connectErr = s.connectIMAP(account)
		return connectErr
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer s.returnIMAP(account.ID, client)

	// For Gmail, emails are synced from "[Gmail]/All Mail" so we use that folder directly
	_, err = client.Select("[Gmail]/All Mail", nil).Wait()
	if err != nil {
		_, err = client.Select(folder.RemoteName, nil).Wait()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to select folder: %w", err)
		}
	}

	uidSet := imap.UIDSetNum(imap.UID(email.UID))

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	messages, err := fetchCmd.Collect()
	if err != nil || len(messages) == 0 {
		return nil, nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	msg := messages[0]
	for _, section := range msg.BodySection {
		if len(section.Bytes) > 0 {
			data, err := s.extractAttachment(section.Bytes, attachment.Filename)
			if err == nil && len(data) > 0 {
				// Cache to MinIO for future downloads
				if s.storage != nil {
					storageKey := fmt.Sprintf("email-attachments/%s/%s/%s", account.ID, email.ID, attachment.Filename)
					uploadErr := s.storage.Upload(ctx, storageKey, bytes.NewReader(data), int64(len(data)), attachment.ContentType)
					if uploadErr == nil {
						// Update attachment with storage key
						s.repo.UpdateAttachmentStorageKey(ctx, attachment.ID, storageKey)
					}
				}
				return attachment, data, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("attachment content not found in message")
}

func (s *EmailService) extractAttachment(body []byte, targetFilename string) ([]byte, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	contentType := msg.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)

	if strings.HasPrefix(mediaType, "multipart/") {
		return s.extractAttachmentFromMultipart(msg.Body, params["boundary"], targetFilename)
	}

	return nil, fmt.Errorf("not a multipart message")
}

func (s *EmailService) extractAttachmentFromMultipart(r io.Reader, boundary string, targetFilename string) ([]byte, error) {
	if boundary == "" {
		return nil, fmt.Errorf("no boundary")
	}

	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}

		contentType := part.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "multipart/") {
			// Recurse into nested multipart
			data, err := s.extractAttachmentFromMultipart(part, params["boundary"], targetFilename)
			if err == nil && len(data) > 0 {
				part.Close()
				return data, nil
			}
		} else if part.FileName() == targetFilename {
			// Found the attachment
			contentTransferEncoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
			bodyBytes, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				return nil, err
			}

			// Decode the content
			switch contentTransferEncoding {
			case "base64":
				s := strings.ReplaceAll(string(bodyBytes), "\r\n", "")
				s = strings.ReplaceAll(s, "\n", "")
				decoded, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return bodyBytes, nil // Return as-is if decode fails
				}
				return decoded, nil
			case "quoted-printable":
				reader := quotedprintable.NewReader(bytes.NewReader(bodyBytes))
				decoded, err := io.ReadAll(reader)
				if err != nil {
					return bodyBytes, nil
				}
				return decoded, nil
			default:
				return bodyBytes, nil
			}
		}
		part.Close()
	}

	return nil, fmt.Errorf("attachment not found")
}

// ============ Labels ============

func (s *EmailService) CreateLabel(ctx context.Context, label *models.EmailLabel) error {
	if label.Color == "" {
		label.Color = "#6B7280" // Default gray
	}
	return s.repo.CreateLabel(ctx, label)
}

func (s *EmailService) GetLabel(ctx context.Context, labelID string) (*models.EmailLabel, error) {
	return s.repo.GetLabelByID(ctx, labelID)
}

func (s *EmailService) GetLabels(ctx context.Context, accountID string) ([]models.EmailLabel, error) {
	return s.repo.GetLabelsByAccount(ctx, accountID)
}

func (s *EmailService) UpdateLabel(ctx context.Context, label *models.EmailLabel) error {
	return s.repo.UpdateLabel(ctx, label)
}

func (s *EmailService) DeleteLabel(ctx context.Context, labelID string) error {
	return s.repo.DeleteLabel(ctx, labelID)
}

func (s *EmailService) AssignLabel(ctx context.Context, emailID, labelID string) error {
	return s.repo.AssignLabelToEmail(ctx, emailID, labelID)
}

func (s *EmailService) RemoveLabel(ctx context.Context, emailID, labelID string) error {
	return s.repo.RemoveLabelFromEmail(ctx, emailID, labelID)
}

func (s *EmailService) GetEmailLabels(ctx context.Context, emailID string) ([]models.EmailLabel, error) {
	return s.repo.GetLabelsByEmail(ctx, emailID)
}

func (s *EmailService) GetEmailsByLabel(ctx context.Context, labelID string, page, pageSize int) ([]models.EmailListItem, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetEmailsByLabel(ctx, labelID, pageSize, offset)
}

// ============ Special Views ============

func (s *EmailService) GetStarredEmails(ctx context.Context, accountID string, page, pageSize int) ([]models.EmailListItem, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetStarredEmails(ctx, accountID, pageSize, offset)
}

func (s *EmailService) GetDraftEmails(ctx context.Context, accountID string, page, pageSize int) ([]models.EmailListItem, error) {
	offset := (page - 1) * pageSize
	return s.repo.GetDraftEmails(ctx, accountID, pageSize, offset)
}

func (s *EmailService) GetStarredCount(ctx context.Context, accountID string) (int, error) {
	return s.repo.GetStarredCount(ctx, accountID)
}

func (s *EmailService) GetDraftCount(ctx context.Context, accountID string) (int, error) {
	return s.repo.GetDraftCount(ctx, accountID)
}

// ============ Rules ============

func (s *EmailService) CreateRule(ctx context.Context, rule *models.EmailRule) error {
	return s.repo.CreateRule(ctx, rule)
}

func (s *EmailService) GetRule(ctx context.Context, ruleID string) (*models.EmailRule, error) {
	return s.repo.GetRuleByID(ctx, ruleID)
}

func (s *EmailService) GetRules(ctx context.Context, accountID string) ([]models.EmailRule, error) {
	return s.repo.GetRulesByAccount(ctx, accountID)
}

func (s *EmailService) UpdateRule(ctx context.Context, rule *models.EmailRule) error {
	return s.repo.UpdateRule(ctx, rule)
}

func (s *EmailService) DeleteRule(ctx context.Context, ruleID string) error {
	return s.repo.DeleteRule(ctx, ruleID)
}

// RunRuleNow applies a specific rule to all existing emails in the account
func (s *EmailService) RunRuleNow(ctx context.Context, ruleID string) (int, error) {
	rule, err := s.repo.GetRuleByID(ctx, ruleID)
	if err != nil {
		return 0, err
	}

	// Get all emails for this account
	emails, err := s.repo.GetAllEmailsForAccount(ctx, rule.AccountID)
	if err != nil {
		return 0, err
	}

	affected := 0
	for _, email := range emails {
		if s.ruleMatches(*rule, &email) {
			for _, action := range rule.Actions {
				if err := s.applyAction(ctx, &email, action); err != nil {
					log.Error().Err(err).Str("action", string(action.Type)).Str("emailID", email.ID).Msg("Error applying rule action")
				}
			}
			affected++
		}
	}

	return affected, nil
}

// ApplyRules applies email rules to an incoming email
func (s *EmailService) ApplyRules(ctx context.Context, email *models.Email) error {
	rules, err := s.repo.GetEnabledRules(ctx, email.AccountID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if s.ruleMatches(rule, email) {
			// Apply all actions
			for _, action := range rule.Actions {
				if err := s.applyAction(ctx, email, action); err != nil {
					// Log error but continue with other actions
					log.Error().Err(err).Str("action", string(action.Type)).Msg("Error applying rule action")
				}
			}

			// Stop processing if configured
			if rule.StopProcessing {
				break
			}
		}
	}

	return nil
}

func (s *EmailService) ruleMatches(rule models.EmailRule, email *models.Email) bool {
	if len(rule.Conditions) == 0 {
		return false
	}

	matchCount := 0
	for _, condition := range rule.Conditions {
		if s.conditionMatches(condition, email) {
			matchCount++
			if rule.MatchType == "any" {
				return true
			}
		}
	}

	// For "all" match type, all conditions must match
	if rule.MatchType == "all" {
		return matchCount == len(rule.Conditions)
	}

	return false
}

func (s *EmailService) conditionMatches(condition models.RuleCondition, email *models.Email) bool {
	var fieldValue string
	switch condition.Field {
	case "from":
		// Check both address and name for from field
		fieldValue = email.FromAddress + " " + email.FromName
	case "to":
		fieldValue = email.ToAddresses
	case "subject":
		fieldValue = email.Subject
	case "body":
		fieldValue = email.TextBody
	default:
		return false
	}

	fieldValueLower := strings.ToLower(fieldValue)
	valueLower := strings.ToLower(condition.Value)

	switch condition.Operator {
	case "contains":
		return strings.Contains(fieldValueLower, valueLower)
	case "equals":
		return fieldValueLower == valueLower
	case "startswith":
		return strings.HasPrefix(fieldValueLower, valueLower)
	case "endswith":
		return strings.HasSuffix(fieldValueLower, valueLower)
	case "regex":
		matched, _ := regexp.MatchString(condition.Value, fieldValue)
		return matched
	default:
		return false
	}
}

func (s *EmailService) applyAction(ctx context.Context, email *models.Email, action models.RuleAction) error {
	switch action.Type {
	case "label":
		if action.Value != "" {
			return s.repo.AssignLabelToEmail(ctx, email.ID, action.Value)
		}
	case "move":
		if action.Value != "" {
			return s.repo.MoveEmail(ctx, email.ID, action.Value)
		}
	case "star":
		return s.repo.MarkAsStarred(ctx, email.ID, true)
	case "mark_read":
		return s.repo.MarkAsRead(ctx, email.ID, true)
	case "delete":
		return s.repo.DeleteEmail(ctx, email.ID)
	}
	return nil
}
