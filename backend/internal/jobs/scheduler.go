package jobs

import (
	"context"
	"log"
	"time"

	"github.com/tessera/tessera/internal/services"
)

// Scheduler handles recurring scheduled jobs
type Scheduler struct {
	worker       *Worker
	emailService *services.EmailService
	stopCh       chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(worker *Worker) *Scheduler {
	return &Scheduler{
		worker: worker,
		stopCh: make(chan struct{}),
	}
}

// SetEmailService sets the email service for email sync scheduling
func (s *Scheduler) SetEmailService(emailService *services.EmailService) {
	s.emailService = emailService
}

// Start begins the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	log.Println("Starting job scheduler")

	go s.scheduleTrashCleanup(ctx)
	go s.scheduleExpiredSharesCleanup(ctx)
	go s.scheduleTempCleanup(ctx)
	go s.scheduleEmailSync(ctx)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// scheduleEmailSync schedules email sync every 30 seconds
func (s *Scheduler) scheduleEmailSync(ctx context.Context) {
	// Wait for system to stabilize
	time.Sleep(time.Second * 15)

	// Run immediately on startup
	s.enqueueEmailSync(ctx)

	// Then run every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.enqueueEmailSync(ctx)
		}
	}
}

func (s *Scheduler) enqueueEmailSync(ctx context.Context) {
	if s.emailService == nil {
		return
	}

	// Get all email accounts
	accounts, err := s.emailService.GetAllAccounts(ctx)
	if err != nil {
		log.Printf("[EMAIL_SYNC] Failed to get accounts: %v", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	log.Printf("[EMAIL_SYNC] Scheduling sync for %d accounts", len(accounts))

	// Enqueue a sync job for each account (skip if already running)
	for _, account := range accounts {
		// Check if sync is already running for this account
		if s.worker.IsJobRunning(JobTypeEmailSync, account.ID) {
			log.Printf("[EMAIL_SYNC] Sync already running for account %s, skipping", account.ID)
			continue
		}
		payload := EmailSyncPayload{
			AccountID: account.ID,
			UserID:    account.UserID,
		}
		if err := s.worker.Enqueue(ctx, JobTypeEmailSync, payload); err != nil {
			log.Printf("[EMAIL_SYNC] Failed to enqueue sync for account %s: %v", account.ID, err)
		}
	}
}

// scheduleTrashCleanup schedules trash cleanup every day
func (s *Scheduler) scheduleTrashCleanup(ctx context.Context) {
	// Run once on startup
	time.Sleep(time.Second * 30) // Wait for system to stabilize
	s.enqueueTrashCleanup(ctx)

	// Then run daily at 3 AM
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.enqueueTrashCleanup(ctx)
		}
	}
}

func (s *Scheduler) enqueueTrashCleanup(ctx context.Context) {
	// Delete items in trash older than 30 days
	payload := CleanupPayload{
		Type:      "trash",
		OlderThan: time.Now().AddDate(0, 0, -30),
	}
	if err := s.worker.Enqueue(ctx, JobTypeCleanup, payload); err != nil {
		log.Printf("Failed to enqueue trash cleanup job: %v", err)
	} else {
		log.Println("Enqueued trash cleanup job")
	}
}

// scheduleExpiredSharesCleanup schedules expired shares cleanup every hour
func (s *Scheduler) scheduleExpiredSharesCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			payload := CleanupPayload{
				Type: "expired_shares",
			}
			if err := s.worker.Enqueue(ctx, JobTypeCleanup, payload); err != nil {
				log.Printf("Failed to enqueue expired shares cleanup job: %v", err)
			}
		}
	}
}

// scheduleTempCleanup schedules temp file cleanup every 6 hours
func (s *Scheduler) scheduleTempCleanup(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			payload := CleanupPayload{
				Type:      "temp",
				OlderThan: time.Now().Add(-6 * time.Hour),
			}
			if err := s.worker.Enqueue(ctx, JobTypeCleanup, payload); err != nil {
				log.Printf("Failed to enqueue temp cleanup job: %v", err)
			}
		}
	}
}

// ScheduleQuotaCheck schedules a quota check for a user
func (s *Scheduler) ScheduleQuotaCheck(ctx context.Context, userID string) error {
	payload := QuotaCheckPayload{
		UserID: userID,
	}
	return s.worker.Enqueue(ctx, JobTypeQuotaCheck, payload)
}

// ScheduleThumbnail schedules thumbnail generation for a file
func (s *Scheduler) ScheduleThumbnail(ctx context.Context, fileID, userID, filePath, mimeType string) error {
	payload := ThumbnailPayload{
		FileID:   fileID,
		UserID:   userID,
		FilePath: filePath,
		MimeType: mimeType,
	}
	return s.worker.Enqueue(ctx, JobTypeThumbnail, payload)
}

// ScheduleNotification schedules a notification for a user
func (s *Scheduler) ScheduleNotification(ctx context.Context, userID, notifType, title, message string, data map[string]interface{}) error {
	payload := NotificationPayload{
		UserID:  userID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    data,
	}
	return s.worker.Enqueue(ctx, JobTypeNotification, payload)
}

// ScheduleVersionCleanup schedules version cleanup for a file
func (s *Scheduler) ScheduleVersionCleanup(ctx context.Context, fileID string, keepVersions int) error {
	payload := VersionCleanupPayload{
		FileID:       fileID,
		KeepVersions: keepVersions,
	}
	return s.worker.Enqueue(ctx, JobTypeVersionCleanup, payload)
}
