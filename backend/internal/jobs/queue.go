package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryQueue is an in-memory implementation of JobQueue
// This is suitable for single-instance deployments
// For production multi-instance setups, use RedisQueue
type MemoryQueue struct {
	jobs      map[string]*Job
	pending   chan *Job
	scheduled []*Job
	mu        sync.RWMutex
}

// NewMemoryQueue creates a new in-memory job queue
func NewMemoryQueue(bufferSize int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	return &MemoryQueue{
		jobs:      make(map[string]*Job),
		pending:   make(chan *Job, bufferSize),
		scheduled: make([]*Job, 0),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	q.jobs[job.ID] = job

	select {
	case q.pending <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Queue is full, job is stored but won't be processed immediately
		return nil
	}
}

// IsJobRunning checks if a job of the given type is currently running or pending for the given key
// For email sync jobs, the key is the account ID
func (q *MemoryQueue) IsJobRunning(jobType JobType, key string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, job := range q.jobs {
		if job.Type != jobType {
			continue
		}
		if job.Status != JobStatusRunning && job.Status != JobStatusPending {
			continue
		}
		// For email sync, check the account ID in the payload
		if jobType == JobTypeEmailSync {
			var payload EmailSyncPayload
			if err := json.Unmarshal(job.Payload, &payload); err == nil {
				if payload.AccountID == key {
					return true
				}
			}
		}
	}
	return false
}

// Dequeue retrieves the next job from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	select {
	case job := <-q.pending:
		q.mu.Lock()
		job.Status = JobStatusRunning
		job.UpdatedAt = time.Now()
		q.mu.Unlock()
		return job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// MarkCompleted marks a job as completed
func (q *MemoryQueue) MarkCompleted(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job, ok := q.jobs[jobID]; ok {
		job.Status = JobStatusCompleted
		job.UpdatedAt = time.Now()
	}
	return nil
}

// MarkFailed marks a job as failed
func (q *MemoryQueue) MarkFailed(ctx context.Context, jobID string, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job, ok := q.jobs[jobID]; ok {
		job.Attempts++
		job.Error = err.Error()
		job.UpdatedAt = time.Now()

		if job.Attempts < job.MaxRetries {
			job.Status = JobStatusRetrying
			// Re-queue the job
			go func() {
				time.Sleep(time.Duration(job.Attempts) * time.Second * 5) // Exponential backoff
				select {
				case q.pending <- job:
				default:
				}
			}()
		} else {
			job.Status = JobStatusFailed
		}
	}
	return nil
}

// Schedule schedules a job to run at a specific time
func (q *MemoryQueue) Schedule(ctx context.Context, job *Job, runAt time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.RunAt = runAt
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	q.jobs[job.ID] = job
	q.scheduled = append(q.scheduled, job)
	return nil
}

// GetJob retrieves a job by ID
func (q *MemoryQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if job, ok := q.jobs[jobID]; ok {
		return job, nil
	}
	return nil, nil
}

// GetPendingJobs retrieves all pending jobs of a specific type
func (q *MemoryQueue) GetPendingJobs(ctx context.Context, jobType JobType) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*Job, 0)
	for _, job := range q.jobs {
		if job.Type == jobType && job.Status == JobStatusPending {
			result = append(result, job)
		}
	}
	return result, nil
}

// ProcessScheduled checks and enqueues scheduled jobs that are ready
func (q *MemoryQueue) ProcessScheduled() {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	remaining := make([]*Job, 0)

	for _, job := range q.scheduled {
		if job.RunAt.Before(now) || job.RunAt.Equal(now) {
			select {
			case q.pending <- job:
			default:
			}
		} else {
			remaining = append(remaining, job)
		}
	}
	q.scheduled = remaining
}

// Stats returns queue statistics
func (q *MemoryQueue) Stats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	statusCounts := make(map[JobStatus]int)
	typeCounts := make(map[JobType]int)

	for _, job := range q.jobs {
		statusCounts[job.Status]++
		typeCounts[job.Type]++
	}

	return map[string]interface{}{
		"total":     len(q.jobs),
		"pending":   len(q.pending),
		"scheduled": len(q.scheduled),
		"by_status": statusCounts,
		"by_type":   typeCounts,
	}
}

// CreateJob is a helper to create a job with a payload
func CreateJob(jobType JobType, payload interface{}) (*Job, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Payload:    data,
		Status:     JobStatusPending,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}
