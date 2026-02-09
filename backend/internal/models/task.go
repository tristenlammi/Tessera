package models

import (
	"time"

	"github.com/google/uuid"
)

// Task represents a task
type Task struct {
	ID                 uuid.UUID       `json:"id"`
	UserID             uuid.UUID       `json:"userId"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	Status             string          `json:"status"`   // todo, in-progress, done
	Priority           string          `json:"priority"` // low, medium, high
	DueDate            *time.Time      `json:"dueDate"`
	GroupID            *uuid.UUID      `json:"groupId"`
	Recurrence         *RecurrenceRule `json:"recurrence"`
	Checklist          []ChecklistItem `json:"checklist"`
	Tags               []string        `json:"tags"`
	Order              int             `json:"order"`
	LinkedEmailID      *string         `json:"linkedEmailId"`
	LinkedEmailSubject *string         `json:"linkedEmailSubject"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	CompletedAt        *time.Time      `json:"completedAt"`
}

// ChecklistItem represents a single item in a task checklist
type ChecklistItem struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	Order     int       `json:"order"`
}

// RecurrenceRule defines task/event recurrence
type RecurrenceRule struct {
	Type                 string     `json:"type"` // daily, weekly, monthly, yearly
	Interval             int        `json:"interval"`
	DaysOfWeek           []int      `json:"daysOfWeek,omitempty"`
	DayOfMonth           *int       `json:"dayOfMonth,omitempty"`
	EndDate              *time.Time `json:"endDate,omitempty"`
	Occurrences          *int       `json:"occurrences,omitempty"`
	OccurrencesCompleted int        `json:"occurrencesCompleted"`
}

// TaskGroup represents a task group
type TaskGroup struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"userId"`
	Name       string          `json:"name"`
	Color      string          `json:"color"`
	Recurrence *RecurrenceRule `json:"recurrence"`
	CreatedAt  time.Time       `json:"createdAt"`
}
