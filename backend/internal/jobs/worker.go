package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Worker processes jobs from the queue
type Worker struct {
	queue       *MemoryQueue
	handlers    map[JobType]JobHandler
	concurrency int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewWorker creates a new job worker
func NewWorker(queue *MemoryQueue, concurrency int) *Worker {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &Worker{
		queue:       queue,
		handlers:    make(map[JobType]JobHandler),
		concurrency: concurrency,
		stopCh:      make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a job type
func (w *Worker) RegisterHandler(jobType JobType, handler JobHandler) {
	w.handlers[jobType] = handler
}

// Start begins processing jobs
func (w *Worker) Start(ctx context.Context) {
	log.Printf("Starting job worker with %d goroutines", w.concurrency)

	// Start worker goroutines
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.process(ctx, i)
	}

	// Start scheduler for scheduled jobs
	w.wg.Add(1)
	go w.scheduler(ctx)
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	log.Println("Stopping job worker...")
	close(w.stopCh)
	w.wg.Wait()
	log.Println("Job worker stopped")
}

func (w *Worker) process(ctx context.Context, id int) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
			// Try to get a job with a timeout
			jobCtx, cancel := context.WithTimeout(ctx, time.Second)
			job, err := w.queue.Dequeue(jobCtx)
			cancel()

			if err != nil || job == nil {
				continue
			}

			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *Job) {
	handler, ok := w.handlers[job.Type]
	if !ok {
		log.Printf("No handler registered for job type: %s", job.Type)
		w.queue.MarkFailed(ctx, job.ID, fmt.Errorf("no handler for job type: %s", job.Type))
		return
	}

	log.Printf("Processing job %s (type: %s, attempt: %d)", job.ID, job.Type, job.Attempts+1)

	// Create a context with timeout for job processing
	// Email sync jobs need longer timeout (30 min) for large mailboxes
	timeout := 5 * time.Minute
	if job.Type == JobTypeEmailSync {
		timeout = 30 * time.Minute
	}
	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := handler.Handle(jobCtx, job)
	if err != nil {
		log.Printf("Job %s failed: %v", job.ID, err)
		w.queue.MarkFailed(ctx, job.ID, err)
	} else {
		log.Printf("Job %s completed successfully", job.ID)
		w.queue.MarkCompleted(ctx, job.ID)
	}
}

func (w *Worker) scheduler(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.queue.ProcessScheduled()
		}
	}
}

// IsJobRunning checks if a job of the given type is currently running for the given key
func (w *Worker) IsJobRunning(jobType JobType, key string) bool {
	return w.queue.IsJobRunning(jobType, key)
}

// Enqueue is a helper to enqueue a job
func (w *Worker) Enqueue(ctx context.Context, jobType JobType, payload interface{}) error {
	job, err := CreateJob(jobType, payload)
	if err != nil {
		return err
	}
	return w.queue.Enqueue(ctx, job)
}

// Schedule is a helper to schedule a job
func (w *Worker) Schedule(ctx context.Context, jobType JobType, payload interface{}, runAt time.Time) error {
	job, err := CreateJob(jobType, payload)
	if err != nil {
		return err
	}
	return w.queue.Schedule(ctx, job, runAt)
}

// ThumbnailHandler handles thumbnail generation jobs
type ThumbnailHandler struct {
	// Add dependencies like storage service
}

func NewThumbnailHandler() *ThumbnailHandler {
	return &ThumbnailHandler{}
}

func (h *ThumbnailHandler) Handle(ctx context.Context, job *Job) error {
	var payload ThumbnailPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Generating thumbnail for file %s", payload.FileID)

	// TODO: Implement thumbnail generation
	// For now, just log and return
	return nil
}

// CleanupHandler handles cleanup jobs
type CleanupHandler struct {
	// Add dependencies
}

func NewCleanupHandler() *CleanupHandler {
	return &CleanupHandler{}
}

func (h *CleanupHandler) Handle(ctx context.Context, job *Job) error {
	var payload CleanupPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Running cleanup job: %s", payload.Type)

	switch payload.Type {
	case "trash":
		// TODO: Clean up old trash items
	case "temp":
		// TODO: Clean up temporary files
	case "expired_shares":
		// TODO: Clean up expired share links
	}

	return nil
}

// NotificationHandler handles notification jobs
type NotificationHandler struct {
	// Add WebSocket hub for real-time notifications
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) Handle(ctx context.Context, job *Job) error {
	var payload NotificationPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Sending notification to user %s: %s", payload.UserID, payload.Title)

	// TODO: Send notification via WebSocket or email
	return nil
}

// QuotaCheckHandler handles quota check jobs
type QuotaCheckHandler struct {
	// Add dependencies
}

func NewQuotaCheckHandler() *QuotaCheckHandler {
	return &QuotaCheckHandler{}
}

func (h *QuotaCheckHandler) Handle(ctx context.Context, job *Job) error {
	var payload QuotaCheckPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Checking quota for user %s", payload.UserID)

	// TODO: Check and update user quota
	return nil
}

// VersionCleanupHandler handles version cleanup jobs
type VersionCleanupHandler struct {
	// Add dependencies
}

func NewVersionCleanupHandler() *VersionCleanupHandler {
	return &VersionCleanupHandler{}
}

func (h *VersionCleanupHandler) Handle(ctx context.Context, job *Job) error {
	var payload VersionCleanupPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Printf("Cleaning up versions for file %s, keeping %d versions", payload.FileID, payload.KeepVersions)

	// TODO: Delete old versions
	return nil
}
