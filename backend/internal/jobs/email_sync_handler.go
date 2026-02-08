package jobs

import (
	"context"
	"encoding/json"
	"log"

	"github.com/tessera/tessera/internal/services"
)

// EmailSyncHandler handles email synchronization jobs
type EmailSyncHandler struct {
	emailService *services.EmailService
}

// NewEmailSyncHandler creates a new email sync handler
func NewEmailSyncHandler(emailService *services.EmailService) *EmailSyncHandler {
	return &EmailSyncHandler{
		emailService: emailService,
	}
}

// Handle processes an email sync job
func (h *EmailSyncHandler) Handle(ctx context.Context, job *Job) error {
	var payload EmailSyncPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}

	log.Printf("[EMAIL_SYNC] Starting sync for account %s", payload.AccountID)

	// Perform the sync
	err := h.emailService.SyncAccount(ctx, payload.AccountID)
	if err != nil {
		log.Printf("[EMAIL_SYNC] Error syncing account %s: %v", payload.AccountID, err)
		return err
	}

	log.Printf("[EMAIL_SYNC] Completed sync for account %s", payload.AccountID)
	return nil
}
