package jobs

import (
	"context"
	"encoding/json"
	"time"
)

// JobType represents different types of background jobs
type JobType string

const (
	JobTypeThumbnail      JobType = "thumbnail"
	JobTypeCleanup        JobType = "cleanup"
	JobTypeNotification   JobType = "notification"
	JobTypeFileIndex      JobType = "file_index"
	JobTypeQuotaCheck     JobType = "quota_check"
	JobTypeVersionCleanup JobType = "version_cleanup"
	JobTypeEmailSync      JobType = "email_sync"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
)

// Job represents a background job
type Job struct {
	ID         string          `json:"id"`
	Type       JobType         `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Status     JobStatus       `json:"status"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	RunAt      time.Time       `json:"run_at,omitempty"`
}

// ThumbnailPayload for thumbnail generation jobs
type ThumbnailPayload struct {
	FileID   string `json:"file_id"`
	UserID   string `json:"user_id"`
	FilePath string `json:"file_path"`
	MimeType string `json:"mime_type"`
}

// CleanupPayload for cleanup jobs
type CleanupPayload struct {
	Type      string    `json:"type"` // "trash", "temp", "expired_shares"
	OlderThan time.Time `json:"older_than,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
}

// NotificationPayload for notification jobs
type NotificationPayload struct {
	UserID  string                 `json:"user_id"`
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// FileIndexPayload for file indexing jobs
type FileIndexPayload struct {
	FileID string `json:"file_id"`
	UserID string `json:"user_id"`
	Action string `json:"action"` // "index", "update", "delete"
}

// QuotaCheckPayload for quota check jobs
type QuotaCheckPayload struct {
	UserID string `json:"user_id"`
}

// VersionCleanupPayload for version cleanup jobs
type VersionCleanupPayload struct {
	FileID       string `json:"file_id"`
	KeepVersions int    `json:"keep_versions"`
}

// EmailSyncPayload for email sync jobs
type EmailSyncPayload struct {
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
}

// JobHandler is the interface for job handlers
type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
}

// JobQueue is the interface for job queue operations
type JobQueue interface {
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context) (*Job, error)
	MarkCompleted(ctx context.Context, jobID string) error
	MarkFailed(ctx context.Context, jobID string, err error) error
	Schedule(ctx context.Context, job *Job, runAt time.Time) error
	GetJob(ctx context.Context, jobID string) (*Job, error)
	GetPendingJobs(ctx context.Context, jobType JobType) ([]*Job, error)
}
