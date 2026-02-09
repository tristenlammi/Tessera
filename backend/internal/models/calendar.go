package models

import (
	"time"

	"github.com/google/uuid"
)

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"userId"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	StartDate    time.Time       `json:"startDate"`
	EndDate      time.Time       `json:"endDate"`
	AllDay       bool            `json:"allDay"`
	Color        string          `json:"color"`
	Recurrence   *RecurrenceRule `json:"recurrence,omitempty"`
	Reminders    []EventReminder `json:"reminders"`
	LinkedTaskID *string         `json:"linkedTaskId,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// EventReminder represents a reminder for an event
type EventReminder struct {
	ID      uuid.UUID `json:"id"`
	Minutes int       `json:"minutes"`
}
