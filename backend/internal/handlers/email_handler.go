package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/services"
)

type EmailHandler struct {
	emailService *services.EmailService
}

func NewEmailHandler(emailService *services.EmailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

// errOwnershipCheck is a sentinel error used by ownership verification helpers.
var errOwnershipCheck = errors.New("ownership check failed")

// verifyAccountOwnership checks that the given account belongs to the authenticated user.
// If verification fails, it sends the appropriate HTTP error response and returns a non-nil error.
// The caller should return nil immediately when a non-nil error is returned (response already sent).
func (h *EmailHandler) verifyAccountOwnership(c *fiber.Ctx, accountID string) error {
	userID := middleware.GetUserID(c).String()
	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
		return errOwnershipCheck
	}
	if account.UserID != userID {
		c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		return errOwnershipCheck
	}
	return nil
}

// verifyFolderOwnership checks that the folder belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyFolderOwnership(c *fiber.Ctx, folderID string) error {
	folder, err := h.emailService.GetFolder(c.Context(), folderID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Folder not found"})
		return errOwnershipCheck
	}
	return h.verifyAccountOwnership(c, folder.AccountID)
}

// verifyEmailOwnership checks that the email belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyEmailOwnership(c *fiber.Ctx, emailID string) error {
	email, err := h.emailService.GetEmail(c.Context(), emailID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Email not found"})
		return errOwnershipCheck
	}
	return h.verifyAccountOwnership(c, email.AccountID)
}

// verifyLabelOwnership checks that the label belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyLabelOwnership(c *fiber.Ctx, labelID string) error {
	label, err := h.emailService.GetLabel(c.Context(), labelID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Label not found"})
		return errOwnershipCheck
	}
	return h.verifyAccountOwnership(c, label.AccountID)
}

// verifyRuleOwnership checks that the rule belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyRuleOwnership(c *fiber.Ctx, ruleID string) error {
	rule, err := h.emailService.GetRule(c.Context(), ruleID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rule not found"})
		return errOwnershipCheck
	}
	return h.verifyAccountOwnership(c, rule.AccountID)
}

// verifyDraftOwnership checks that the draft belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyDraftOwnership(c *fiber.Ctx, draftID string) error {
	draft, err := h.emailService.GetDraft(c.Context(), draftID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Draft not found"})
		return errOwnershipCheck
	}
	return h.verifyAccountOwnership(c, draft.AccountID)
}

// verifyAttachmentOwnership checks that the attachment belongs to an account owned by the authenticated user.
func (h *EmailHandler) verifyAttachmentOwnership(c *fiber.Ctx, attachmentID string) error {
	attachment, err := h.emailService.GetAttachment(c.Context(), attachmentID)
	if err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Attachment not found"})
		return errOwnershipCheck
	}
	return h.verifyEmailOwnership(c, attachment.EmailID)
}

// ============ Email Accounts ============

type CreateAccountInput struct {
	Name         string `json:"name"`
	EmailAddress string `json:"email_address"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	IMAPUsername string `json:"imap_username"`
	IMAPPassword string `json:"imap_password"`
	IMAPUseTLS   bool   `json:"imap_use_tls"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
	IsDefault    bool   `json:"is_default"`
}

func (h *EmailHandler) CreateAccount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()

	var input CreateAccountInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if input.Name == "" || input.EmailAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and email address are required"})
	}
	if input.IMAPHost == "" || input.IMAPUsername == "" || input.IMAPPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "IMAP settings are required"})
	}
	if input.SMTPHost == "" || input.SMTPUsername == "" || input.SMTPPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "SMTP settings are required"})
	}

	// Set defaults
	if input.IMAPPort == 0 {
		input.IMAPPort = 993
	}
	if input.SMTPPort == 0 {
		input.SMTPPort = 587
	}

	account := &models.EmailAccount{
		UserID:       userID,
		Name:         input.Name,
		EmailAddress: input.EmailAddress,
		IMAPHost:     input.IMAPHost,
		IMAPPort:     input.IMAPPort,
		IMAPUsername: input.IMAPUsername,
		IMAPPassword: input.IMAPPassword,
		IMAPUseTLS:   input.IMAPUseTLS,
		SMTPHost:     input.SMTPHost,
		SMTPPort:     input.SMTPPort,
		SMTPUsername: input.SMTPUsername,
		SMTPPassword: input.SMTPPassword,
		SMTPUseTLS:   input.SMTPUseTLS,
		IsDefault:    input.IsDefault,
	}

	if err := h.emailService.CreateAccount(c.Context(), account); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Strip sensitive data from response
	account.IMAPPassword = ""
	account.SMTPPassword = ""

	return c.Status(fiber.StatusCreated).JSON(account)
}

func (h *EmailHandler) GetAccounts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()

	accounts, err := h.emailService.GetAccounts(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get accounts"})
	}

	if accounts == nil {
		accounts = []models.EmailAccount{}
	}

	// Strip sensitive data from response
	for i := range accounts {
		accounts[i].IMAPPassword = ""
		accounts[i].SMTPPassword = ""
	}

	return c.JSON(accounts)
}

func (h *EmailHandler) GetAccount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()
	accountID := c.Params("accountId")

	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}

	// Verify ownership
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	// Strip sensitive data from response
	account.IMAPPassword = ""
	account.SMTPPassword = ""

	return c.JSON(account)
}

func (h *EmailHandler) UpdateAccount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()
	accountID := c.Params("accountId")

	var input CreateAccountInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}

	// Verify ownership
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	// Update fields
	account.Name = input.Name
	account.EmailAddress = input.EmailAddress
	account.IMAPHost = input.IMAPHost
	account.IMAPPort = input.IMAPPort
	account.IMAPUsername = input.IMAPUsername
	if input.IMAPPassword != "" {
		account.IMAPPassword = input.IMAPPassword
	}
	account.IMAPUseTLS = input.IMAPUseTLS
	account.SMTPHost = input.SMTPHost
	account.SMTPPort = input.SMTPPort
	account.SMTPUsername = input.SMTPUsername
	if input.SMTPPassword != "" {
		account.SMTPPassword = input.SMTPPassword
	}
	account.SMTPUseTLS = input.SMTPUseTLS
	account.IsDefault = input.IsDefault

	if err := h.emailService.UpdateAccount(c.Context(), account); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update account"})
	}

	// Strip sensitive data from response
	account.IMAPPassword = ""
	account.SMTPPassword = ""

	return c.JSON(account)
}

func (h *EmailHandler) DeleteAccount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()
	accountID := c.Params("accountId")

	// Verify ownership before delete
	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	if err := h.emailService.DeleteAccount(c.Context(), accountID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete account"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *EmailHandler) SyncAccount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()
	accountID := c.Params("accountId")

	// Verify ownership before sync
	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	// Use a background context so sync continues even if HTTP request times out
	// This is a long-running operation that can take several minutes
	ctx := context.Background()

	err = h.emailService.SyncAccount(ctx, accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Sync completed"})
}

func (h *EmailHandler) SyncAccountStream(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c).String()
	accountID := c.Params("accountId")

	// Verify ownership before sync
	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	ctx := c.Context()

	// Create a progress channel
	progressChan := make(chan map[string]interface{}, 100)

	// Run sync with progress callback
	go func() {
		h.emailService.SyncAccountWithProgress(context.Background(), accountID, func(progress map[string]interface{}) {
			select {
			case progressChan <- progress:
			default:
				// Channel full, skip this update
			}
		})
		close(progressChan)
	}()

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		for progress := range progressChan {
			data, _ := json.Marshal(progress)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
		}
	})

	return nil
}

// ============ Email Folders ============

func (h *EmailHandler) GetFolders(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	folders, err := h.emailService.GetFolders(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get folders"})
	}

	if folders == nil {
		folders = []models.EmailFolder{}
	}

	return c.JSON(folders)
}

func (h *EmailHandler) GetFoldersTree(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	folders, err := h.emailService.GetFoldersAsTree(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get folders"})
	}

	if folders == nil {
		folders = []*models.EmailFolder{}
	}

	return c.JSON(folders)
}

func (h *EmailHandler) CreateFolder(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Folder name is required"})
	}

	folder := &models.EmailFolder{
		AccountID: accountID,
		Name:      input.Name,
		ParentID:  input.ParentID,
	}

	if err := h.emailService.CreateFolder(c.Context(), folder); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(folder)
}

func (h *EmailHandler) UpdateFolder(c *fiber.Ctx) error {
	folderID := c.Params("folderId")

	var input struct {
		Name      string  `json:"name"`
		ParentID  *string `json:"parent_id"`
		SortOrder *int    `json:"sort_order"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	folder, err := h.emailService.GetFolder(c.Context(), folderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Folder not found"})
	}
	if err := h.verifyAccountOwnership(c, folder.AccountID); err != nil {
		return nil
	}

	if input.Name != "" {
		folder.Name = input.Name
	}
	if input.ParentID != nil {
		folder.ParentID = input.ParentID
	}
	if input.SortOrder != nil {
		folder.SortOrder = *input.SortOrder
	}

	if err := h.emailService.UpdateFolder(c.Context(), folder); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(folder)
}

func (h *EmailHandler) DeleteFolder(c *fiber.Ctx) error {
	folderID := c.Params("folderId")
	if err := h.verifyFolderOwnership(c, folderID); err != nil {
		return nil
	}

	if err := h.emailService.DeleteFolder(c.Context(), folderID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *EmailHandler) MoveFolder(c *fiber.Ctx) error {
	folderID := c.Params("folderId")
	if err := h.verifyFolderOwnership(c, folderID); err != nil {
		return nil
	}

	var input struct {
		ParentID *string `json:"parent_id"`
		TargetID *string `json:"target_id"`
		Position *string `json:"position"` // "before" or "after"
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// If position is specified, reorder relative to target
	if input.Position != nil && input.TargetID != nil {
		if err := h.emailService.ReorderFolderRelative(c.Context(), folderID, *input.TargetID, *input.Position); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	} else {
		// Move to new parent (nesting)
		if err := h.emailService.MoveFolder(c.Context(), folderID, input.ParentID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *EmailHandler) ReorderFolders(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		Orders []struct {
			ID        string `json:"id"`
			SortOrder int    `json:"sort_order"`
		} `json:"orders"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var orders []struct {
		ID        string
		SortOrder int
	}
	for _, o := range input.Orders {
		orders = append(orders, struct {
			ID        string
			SortOrder int
		}{o.ID, o.SortOrder})
	}

	if err := h.emailService.ReorderFolders(c.Context(), orders); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

// ============ Emails ============

func (h *EmailHandler) GetEmails(c *fiber.Ctx) error {
	folderID := c.Params("folderId")
	if err := h.verifyFolderOwnership(c, folderID); err != nil {
		return nil
	}
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 50)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	emails, err := h.emailService.GetEmails(c.Context(), folderID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get emails"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	return c.JSON(emails)
}

// GetThreads returns conversation threads for a folder
func (h *EmailHandler) GetThreads(c *fiber.Ctx) error {
	folderID := c.Params("folderId")
	if err := h.verifyFolderOwnership(c, folderID); err != nil {
		return nil
	}
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 50)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	threads, total, err := h.emailService.GetThreads(c.Context(), folderID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get threads"})
	}

	if threads == nil {
		threads = []models.EmailThread{}
	}

	return c.JSON(fiber.Map{
		"threads":  threads,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetThreadEmails returns all emails in a specific thread
func (h *EmailHandler) GetThreadEmails(c *fiber.Ctx) error {
	threadID := c.Params("threadId")

	emails, err := h.emailService.GetThreadEmails(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get thread emails"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	// Verify ownership via the first email in the thread
	if len(emails) > 0 {
		if err := h.verifyEmailOwnership(c, emails[0].ID); err != nil {
			return nil
		}
	}

	return c.JSON(emails)
}

// GetThreadConversation returns full email objects for all emails in a thread (with bodies)
func (h *EmailHandler) GetThreadConversation(c *fiber.Ctx) error {
	threadID := c.Params("threadId")

	emails, err := h.emailService.GetThreadConversation(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get thread conversation"})
	}

	if emails == nil {
		emails = []models.Email{}
	}

	// Verify ownership via the first email in the thread
	if len(emails) > 0 {
		if err := h.verifyAccountOwnership(c, emails[0].AccountID); err != nil {
			return nil
		}
	}

	// Collect unread email IDs and mark as read in background
	var unreadIDs []string
	for i := range emails {
		if !emails[i].IsRead {
			unreadIDs = append(unreadIDs, emails[i].ID)
			emails[i].IsRead = true // Reflect in response immediately
		}
	}
	if len(unreadIDs) > 0 {
		go h.emailService.BatchMarkAsRead(context.Background(), unreadIDs, true)
	}

	return c.JSON(emails)
}

// ReindexThreads rebuilds thread IDs for existing emails
func (h *EmailHandler) ReindexThreads(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	if err := h.emailService.ReindexThreads(c.Context(), accountID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reindex threads"})
	}

	return c.JSON(fiber.Map{"message": "Thread reindexing completed"})
}

func (h *EmailHandler) GetEmail(c *fiber.Ctx) error {
	emailID := c.Params("emailId")

	email, err := h.emailService.GetEmail(c.Context(), emailID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Email not found"})
	}
	if err := h.verifyAccountOwnership(c, email.AccountID); err != nil {
		return nil
	}

	// Mark as read in background (don't block the response)
	if !email.IsRead {
		go h.emailService.MarkAsRead(context.Background(), emailID, true)
		email.IsRead = true // Reflect in response immediately
	}

	return c.JSON(email)
}

func (h *EmailHandler) MarkAsRead(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	var input struct {
		IsRead bool `json:"is_read"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.emailService.MarkAsRead(c.Context(), emailID, input.IsRead); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update email"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) MarkFolderAsRead(c *fiber.Ctx) error {
	folderID := c.Params("folderId")
	if err := h.verifyFolderOwnership(c, folderID); err != nil {
		return nil
	}
	log.Printf("[MARK_READ] Marking folder %s as read", folderID)

	count, err := h.emailService.MarkFolderAsRead(c.Context(), folderID)
	if err != nil {
		log.Printf("[MARK_READ] Error marking folder as read: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark folder as read"})
	}

	log.Printf("[MARK_READ] Successfully marked %d emails as read in folder %s", count, folderID)
	return c.JSON(fiber.Map{"success": true, "updated": count})
}

func (h *EmailHandler) MarkAsStarred(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	var input struct {
		IsStarred bool `json:"is_starred"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.emailService.MarkAsStarred(c.Context(), emailID, input.IsStarred); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update email"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) MoveEmail(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	var input struct {
		FolderID string `json:"folder_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.emailService.MoveEmail(c.Context(), emailID, input.FolderID); err != nil {
		log.Printf("MoveEmail error: emailID=%s folder=%s err=%v", emailID, input.FolderID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to move email"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) DeleteEmail(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	if err := h.emailService.DeleteEmail(c.Context(), emailID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete email"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *EmailHandler) SearchEmails(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}
	query := c.Query("q")

	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Search query is required"})
	}

	// Use advanced search if operators are detected
	emails, err := h.emailService.AdvancedSearchEmails(c.Context(), accountID, query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Search failed"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	return c.JSON(emails)
}

// ============ Batch Operations ============

func (h *EmailHandler) BatchMarkAsRead(c *fiber.Ctx) error {
	var input struct {
		EmailIDs []string `json:"email_ids"`
		IsRead   bool     `json:"is_read"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(input.EmailIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No email IDs provided"})
	}
	for _, eid := range input.EmailIDs {
		if err := h.verifyEmailOwnership(c, eid); err != nil {
			return nil
		}
	}

	count, err := h.emailService.BatchMarkAsRead(c.Context(), input.EmailIDs, input.IsRead)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update emails"})
	}

	return c.JSON(fiber.Map{"success": true, "updated": count})
}

func (h *EmailHandler) BatchMarkAsStarred(c *fiber.Ctx) error {
	var input struct {
		EmailIDs  []string `json:"email_ids"`
		IsStarred bool     `json:"is_starred"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(input.EmailIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No email IDs provided"})
	}
	for _, eid := range input.EmailIDs {
		if err := h.verifyEmailOwnership(c, eid); err != nil {
			return nil
		}
	}

	count, err := h.emailService.BatchMarkAsStarred(c.Context(), input.EmailIDs, input.IsStarred)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update emails"})
	}

	return c.JSON(fiber.Map{"success": true, "updated": count})
}

func (h *EmailHandler) BatchMoveEmails(c *fiber.Ctx) error {
	var input struct {
		EmailIDs []string `json:"email_ids"`
		FolderID string   `json:"folder_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(input.EmailIDs) == 0 || input.FolderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email IDs and folder ID are required"})
	}
	for _, eid := range input.EmailIDs {
		if err := h.verifyEmailOwnership(c, eid); err != nil {
			return nil
		}
	}

	count, err := h.emailService.BatchMoveEmails(c.Context(), input.EmailIDs, input.FolderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to move emails"})
	}

	return c.JSON(fiber.Map{"success": true, "moved": count})
}

func (h *EmailHandler) BatchDeleteEmails(c *fiber.Ctx) error {
	var input struct {
		EmailIDs []string `json:"email_ids"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(input.EmailIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No email IDs provided"})
	}
	for _, eid := range input.EmailIDs {
		if err := h.verifyEmailOwnership(c, eid); err != nil {
			return nil
		}
	}

	count, err := h.emailService.BatchDeleteEmails(c.Context(), input.EmailIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete emails"})
	}

	return c.JSON(fiber.Map{"success": true, "deleted": count})
}

func (h *EmailHandler) BatchAssignLabel(c *fiber.Ctx) error {
	var input struct {
		EmailIDs []string `json:"email_ids"`
		LabelID  string   `json:"label_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if len(input.EmailIDs) == 0 || input.LabelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email IDs and label ID are required"})
	}
	for _, eid := range input.EmailIDs {
		if err := h.verifyEmailOwnership(c, eid); err != nil {
			return nil
		}
	}

	count, err := h.emailService.BatchAssignLabel(c.Context(), input.EmailIDs, input.LabelID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign label"})
	}

	return c.JSON(fiber.Map{"success": true, "assigned": count})
}

// ============ Compose Drafts ============

func (h *EmailHandler) SaveDraft(c *fiber.Ctx) error {
	var input struct {
		ID        string                `json:"id"`
		AccountID string                `json:"account_id"`
		To        []models.EmailAddress `json:"to"`
		CC        []models.EmailAddress `json:"cc"`
		BCC       []models.EmailAddress `json:"bcc"`
		Subject   string                `json:"subject"`
		Body      string                `json:"body"`
		IsHTML    bool                  `json:"is_html"`
		ReplyToID string                `json:"reply_to_id"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.verifyAccountOwnership(c, input.AccountID); err != nil {
		return nil
	}

	draft := &models.EmailDraft{
		ID:        input.ID,
		AccountID: input.AccountID,
		To:        input.To,
		CC:        input.CC,
		BCC:       input.BCC,
		Subject:   input.Subject,
		Body:      input.Body,
		IsHTML:    input.IsHTML,
		ReplyToID: input.ReplyToID,
	}

	if err := h.emailService.SaveDraft(c.Context(), draft); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save draft"})
	}

	return c.JSON(draft)
}

func (h *EmailHandler) GetDrafts(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	drafts, err := h.emailService.GetDrafts(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get drafts"})
	}

	if drafts == nil {
		drafts = []models.EmailDraft{}
	}

	return c.JSON(drafts)
}

func (h *EmailHandler) GetDraft(c *fiber.Ctx) error {
	draftID := c.Params("draftId")
	if err := h.verifyDraftOwnership(c, draftID); err != nil {
		return nil
	}

	draft, err := h.emailService.GetDraft(c.Context(), draftID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Draft not found"})
	}

	return c.JSON(draft)
}

func (h *EmailHandler) DeleteDraft(c *fiber.Ctx) error {
	draftID := c.Params("draftId")
	if err := h.verifyDraftOwnership(c, draftID); err != nil {
		return nil
	}

	if err := h.emailService.DeleteDraft(c.Context(), draftID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete draft"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============ Undo Send ============

func (h *EmailHandler) QueueSend(c *fiber.Ctx) error {
	var input SendEmailInput

	contentType := string(c.Request().Header.ContentType())

	if strings.HasPrefix(contentType, "multipart/form-data") {
		input.AccountID = c.FormValue("account_id")
		input.Subject = c.FormValue("subject")
		input.Body = c.FormValue("body")
		input.IsHTML = c.FormValue("is_html") == "true"
		input.ReplyToID = c.FormValue("reply_to")
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid multipart form"})
		}
		input.To = form.Value["to"]
		input.CC = form.Value["cc"]
		input.BCC = form.Value["bcc"]
	} else {
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
	}

	if input.AccountID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Account ID is required"})
	}
	if len(input.To) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "At least one recipient is required"})
	}

	compose := &models.ComposeEmail{
		AccountID: input.AccountID,
		Subject:   input.Subject,
		Body:      input.Body,
		IsHTML:    input.IsHTML,
		ReplyToID: input.ReplyToID,
	}
	for _, addr := range input.To {
		compose.To = append(compose.To, models.EmailAddress{Address: addr})
	}
	for _, addr := range input.CC {
		compose.CC = append(compose.CC, models.EmailAddress{Address: addr})
	}
	for _, addr := range input.BCC {
		compose.BCC = append(compose.BCC, models.EmailAddress{Address: addr})
	}

	// Handle file attachments from multipart form
	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, _ := c.MultipartForm()
		if form != nil && form.File["attachments"] != nil {
			for _, fh := range form.File["attachments"] {
				file, err := fh.Open()
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to read attachment %s: %v", fh.Filename, err)})
				}
				data, err := io.ReadAll(file)
				file.Close()
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to read attachment %s: %v", fh.Filename, err)})
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" {
					ct = "application/octet-stream"
				}
				compose.FileAttachments = append(compose.FileAttachments, models.FileAttachment{
					Filename:    fh.Filename,
					ContentType: ct,
					Data:        data,
				})
			}
		}
	}

	// Get account to check send_delay
	account, err := h.emailService.GetAccount(c.Context(), input.AccountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}
	userID := middleware.GetUserID(c).String()
	if account.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
	}

	delay := account.SendDelay
	if delay <= 0 {
		// No delay, send immediately
		if err := h.emailService.SendEmail(c.Context(), compose.AccountID, compose); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "message": "Email sent successfully", "immediate": true})
	}

	sendID, err := h.emailService.QueueSend(c.Context(), input.AccountID, compose, delay)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "send_id": sendID, "delay": delay, "message": "Email queued"})
}

func (h *EmailHandler) CancelSend(c *fiber.Ctx) error {
	sendID := c.Params("sendId")

	if err := h.emailService.CancelSend(c.Context(), sendID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Send cancelled"})
}

// ============ Account Settings ============

func (h *EmailHandler) UpdateSignature(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		Signature string `json:"signature"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}

	account.Signature = input.Signature
	if err := h.emailService.UpdateAccount(c.Context(), account); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update signature"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) UpdateSendDelay(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		SendDelay int `json:"send_delay"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.SendDelay < 0 || input.SendDelay > 30 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Send delay must be between 0 and 30 seconds"})
	}

	account, err := h.emailService.GetAccount(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Account not found"})
	}

	account.SendDelay = input.SendDelay
	if err := h.emailService.UpdateAccount(c.Context(), account); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update send delay"})
	}

	return c.JSON(fiber.Map{"success": true})
}

// ============ Send Email ============

type SendEmailInput struct {
	AccountID string   `json:"account_id"`
	To        []string `json:"to"`
	CC        []string `json:"cc"`
	BCC       []string `json:"bcc"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	IsHTML    bool     `json:"is_html"`
	ReplyToID string   `json:"reply_to"`
}

func (h *EmailHandler) SendEmail(c *fiber.Ctx) error {
	var input SendEmailInput

	contentType := string(c.Request().Header.ContentType())

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Parse from multipart form
		input.AccountID = c.FormValue("account_id")
		input.Subject = c.FormValue("subject")
		input.Body = c.FormValue("body")
		input.IsHTML = c.FormValue("is_html") == "true"
		input.ReplyToID = c.FormValue("reply_to")

		// Fiber's MultipartForm gives us repeated field values
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid multipart form"})
		}
		input.To = form.Value["to"]
		input.CC = form.Value["cc"]
		input.BCC = form.Value["bcc"]
	} else {
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
	}

	if input.AccountID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Account ID is required"})
	}
	if len(input.To) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "At least one recipient is required"})
	}
	if err := h.verifyAccountOwnership(c, input.AccountID); err != nil {
		return nil
	}

	// Convert string addresses to EmailAddress structs
	compose := &models.ComposeEmail{
		AccountID: input.AccountID,
		Subject:   input.Subject,
		Body:      input.Body,
		IsHTML:    input.IsHTML,
		ReplyToID: input.ReplyToID,
	}

	for _, addr := range input.To {
		compose.To = append(compose.To, models.EmailAddress{Address: addr})
	}
	for _, addr := range input.CC {
		compose.CC = append(compose.CC, models.EmailAddress{Address: addr})
	}
	for _, addr := range input.BCC {
		compose.BCC = append(compose.BCC, models.EmailAddress{Address: addr})
	}

	// Handle file attachments from multipart form
	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, _ := c.MultipartForm()
		if form != nil && form.File["attachments"] != nil {
			for _, fh := range form.File["attachments"] {
				file, err := fh.Open()
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to read attachment %s: %v", fh.Filename, err)})
				}
				data, err := io.ReadAll(file)
				file.Close()
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to read attachment %s: %v", fh.Filename, err)})
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" {
					ct = "application/octet-stream"
				}
				compose.FileAttachments = append(compose.FileAttachments, models.FileAttachment{
					Filename:    fh.Filename,
					ContentType: ct,
					Data:        data,
				})
			}
		}
	}

	if err := h.emailService.SendEmail(c.Context(), compose.AccountID, compose); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Email sent successfully"})
}

// ============ Labels ============

func (h *EmailHandler) CreateLabel(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Label name is required"})
	}

	label := &models.EmailLabel{
		AccountID: accountID,
		Name:      input.Name,
		Color:     input.Color,
	}

	if err := h.emailService.CreateLabel(c.Context(), label); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create label"})
	}

	return c.Status(fiber.StatusCreated).JSON(label)
}

func (h *EmailHandler) GetLabels(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	labels, err := h.emailService.GetLabels(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get labels"})
	}

	if labels == nil {
		labels = []models.EmailLabel{}
	}

	return c.JSON(labels)
}

func (h *EmailHandler) UpdateLabel(c *fiber.Ctx) error {
	labelID := c.Params("labelId")

	label, err := h.emailService.GetLabel(c.Context(), labelID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Label not found"})
	}
	if err := h.verifyAccountOwnership(c, label.AccountID); err != nil {
		return nil
	}

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Name != "" {
		label.Name = input.Name
	}
	if input.Color != "" {
		label.Color = input.Color
	}

	if err := h.emailService.UpdateLabel(c.Context(), label); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update label"})
	}

	return c.JSON(label)
}

func (h *EmailHandler) DeleteLabel(c *fiber.Ctx) error {
	labelID := c.Params("labelId")
	if err := h.verifyLabelOwnership(c, labelID); err != nil {
		return nil
	}

	if err := h.emailService.DeleteLabel(c.Context(), labelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete label"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *EmailHandler) AssignLabel(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	labelID := c.Params("labelId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	if err := h.emailService.AssignLabel(c.Context(), emailID, labelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign label"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) RemoveLabel(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	labelID := c.Params("labelId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	if err := h.emailService.RemoveLabel(c.Context(), emailID, labelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove label"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *EmailHandler) GetEmailLabels(c *fiber.Ctx) error {
	emailID := c.Params("emailId")
	if err := h.verifyEmailOwnership(c, emailID); err != nil {
		return nil
	}

	labels, err := h.emailService.GetEmailLabels(c.Context(), emailID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get email labels"})
	}

	if labels == nil {
		labels = []models.EmailLabel{}
	}

	return c.JSON(labels)
}

func (h *EmailHandler) GetEmailsByLabel(c *fiber.Ctx) error {
	labelID := c.Params("labelId")
	if err := h.verifyLabelOwnership(c, labelID); err != nil {
		return nil
	}
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("limit", 50)

	emails, err := h.emailService.GetEmailsByLabel(c.Context(), labelID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get emails"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	return c.JSON(emails)
}

// ============ Special Views ============

func (h *EmailHandler) GetStarredEmails(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("limit", 50)

	emails, err := h.emailService.GetStarredEmails(c.Context(), accountID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get starred emails"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	return c.JSON(emails)
}

func (h *EmailHandler) GetDraftEmails(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("limit", 50)

	emails, err := h.emailService.GetDraftEmails(c.Context(), accountID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get draft emails"})
	}

	if emails == nil {
		emails = []models.EmailListItem{}
	}

	return c.JSON(emails)
}

func (h *EmailHandler) GetCounts(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	starredCount, _ := h.emailService.GetStarredCount(c.Context(), accountID)
	draftCount, _ := h.emailService.GetDraftCount(c.Context(), accountID)

	return c.JSON(fiber.Map{
		"starred": starredCount,
		"drafts":  draftCount,
	})
}

// ============ Rules ============

func (h *EmailHandler) CreateRule(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	var input struct {
		Name           string                 `json:"name"`
		IsEnabled      bool                   `json:"is_enabled"`
		Priority       int                    `json:"priority"`
		MatchType      string                 `json:"match_type"`
		Conditions     []models.RuleCondition `json:"conditions"`
		Actions        []models.RuleAction    `json:"actions"`
		StopProcessing bool                   `json:"stop_processing"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Rule name is required"})
	}
	if len(input.Conditions) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "At least one condition is required"})
	}
	if len(input.Actions) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "At least one action is required"})
	}

	rule := &models.EmailRule{
		AccountID:      accountID,
		Name:           input.Name,
		IsEnabled:      input.IsEnabled,
		Priority:       input.Priority,
		MatchType:      input.MatchType,
		Conditions:     input.Conditions,
		Actions:        input.Actions,
		StopProcessing: input.StopProcessing,
	}

	if rule.MatchType == "" {
		rule.MatchType = "any"
	}

	if err := h.emailService.CreateRule(c.Context(), rule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create rule"})
	}

	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (h *EmailHandler) GetRules(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if err := h.verifyAccountOwnership(c, accountID); err != nil {
		return nil
	}

	rules, err := h.emailService.GetRules(c.Context(), accountID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get rules"})
	}

	if rules == nil {
		rules = []models.EmailRule{}
	}

	return c.JSON(rules)
}

func (h *EmailHandler) UpdateRule(c *fiber.Ctx) error {
	ruleID := c.Params("ruleId")

	rule, err := h.emailService.GetRule(c.Context(), ruleID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rule not found"})
	}
	if err := h.verifyAccountOwnership(c, rule.AccountID); err != nil {
		return nil
	}

	var input struct {
		Name           string                 `json:"name"`
		IsEnabled      *bool                  `json:"is_enabled"`
		Priority       *int                   `json:"priority"`
		MatchType      string                 `json:"match_type"`
		Conditions     []models.RuleCondition `json:"conditions"`
		Actions        []models.RuleAction    `json:"actions"`
		StopProcessing *bool                  `json:"stop_processing"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Name != "" {
		rule.Name = input.Name
	}
	if input.IsEnabled != nil {
		rule.IsEnabled = *input.IsEnabled
	}
	if input.Priority != nil {
		rule.Priority = *input.Priority
	}
	if input.MatchType != "" {
		rule.MatchType = input.MatchType
	}
	if input.Conditions != nil {
		rule.Conditions = input.Conditions
	}
	if input.Actions != nil {
		rule.Actions = input.Actions
	}
	if input.StopProcessing != nil {
		rule.StopProcessing = *input.StopProcessing
	}

	if err := h.emailService.UpdateRule(c.Context(), rule); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update rule"})
	}

	return c.JSON(rule)
}

func (h *EmailHandler) DeleteRule(c *fiber.Ctx) error {
	ruleID := c.Params("ruleId")
	if err := h.verifyRuleOwnership(c, ruleID); err != nil {
		return nil
	}

	if err := h.emailService.DeleteRule(c.Context(), ruleID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete rule"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RunRule applies a rule to all existing emails in the account
func (h *EmailHandler) RunRule(c *fiber.Ctx) error {
	ruleID := c.Params("ruleId")
	if err := h.verifyRuleOwnership(c, ruleID); err != nil {
		return nil
	}

	affected, err := h.emailService.RunRuleNow(c.Context(), ruleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to run rule"})
	}

	return c.JSON(fiber.Map{
		"affected": affected,
		"message":  fmt.Sprintf("Rule applied to %d emails", affected),
	})
}

// ============ Attachments ============

func (h *EmailHandler) GetAttachment(c *fiber.Ctx) error {
	attachmentID := c.Params("attachmentId")
	if err := h.verifyAttachmentOwnership(c, attachmentID); err != nil {
		return nil
	}

	attachment, err := h.emailService.GetAttachment(c.Context(), attachmentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Attachment not found"})
	}

	return c.JSON(attachment)
}

func (h *EmailHandler) DownloadAttachment(c *fiber.Ctx) error {
	attachmentID := c.Params("attachmentId")
	if err := h.verifyAttachmentOwnership(c, attachmentID); err != nil {
		return nil
	}

	attachment, data, err := h.emailService.DownloadAttachment(c.Context(), attachmentID)
	if err != nil {
		log.Printf("Error downloading attachment: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to download attachment"})
	}

	// Sanitize filename to prevent header injection
	safeName := strings.ReplaceAll(attachment.Filename, "\"", "")
	safeName = strings.ReplaceAll(safeName, "\n", "")
	safeName = strings.ReplaceAll(safeName, "\r", "")

	// Set headers for download
	c.Set("Content-Type", attachment.ContentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", safeName))
	c.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	return c.Send(data)
}
