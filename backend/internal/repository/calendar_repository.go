package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

// CalendarRepository handles calendar event database operations
type CalendarRepository struct {
	db *pgxpool.Pool
}

// NewCalendarRepository creates a new calendar repository
func NewCalendarRepository(db *pgxpool.Pool) *CalendarRepository {
	return &CalendarRepository{db: db}
}

// ListByUser returns events for a user within a date range
func (r *CalendarRepository) ListByUser(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]models.CalendarEvent, error) {
	query := `
		SELECT id, user_id, title, description, start_date, end_date, all_day,
		       color, recurrence, reminders, linked_task_id, created_at, updated_at
		FROM calendar_events
		WHERE user_id = $1 AND start_date <= $3 AND end_date >= $2
		ORDER BY start_date ASC
	`

	rows, err := r.db.Query(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.CalendarEvent
	for rows.Next() {
		var e models.CalendarEvent
		var recurrenceJSON, remindersJSON []byte
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.Title, &e.Description, &e.StartDate, &e.EndDate,
			&e.AllDay, &e.Color, &recurrenceJSON, &remindersJSON, &e.LinkedTaskID,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if recurrenceJSON != nil {
			var rec models.RecurrenceRule
			if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
				e.Recurrence = &rec
			}
		}
		if remindersJSON != nil {
			json.Unmarshal(remindersJSON, &e.Reminders)
		}
		if e.Reminders == nil {
			e.Reminders = []models.EventReminder{}
		}
		events = append(events, e)
	}

	if events == nil {
		events = []models.CalendarEvent{}
	}
	return events, nil
}

// GetByID returns a single event
func (r *CalendarRepository) GetByID(ctx context.Context, eventID, userID uuid.UUID) (*models.CalendarEvent, error) {
	query := `
		SELECT id, user_id, title, description, start_date, end_date, all_day,
		       color, recurrence, reminders, linked_task_id, created_at, updated_at
		FROM calendar_events
		WHERE id = $1 AND user_id = $2
	`

	var e models.CalendarEvent
	var recurrenceJSON, remindersJSON []byte
	err := r.db.QueryRow(ctx, query, eventID, userID).Scan(
		&e.ID, &e.UserID, &e.Title, &e.Description, &e.StartDate, &e.EndDate,
		&e.AllDay, &e.Color, &recurrenceJSON, &remindersJSON, &e.LinkedTaskID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if recurrenceJSON != nil {
		var rec models.RecurrenceRule
		if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
			e.Recurrence = &rec
		}
	}
	if remindersJSON != nil {
		json.Unmarshal(remindersJSON, &e.Reminders)
	}
	if e.Reminders == nil {
		e.Reminders = []models.EventReminder{}
	}
	return &e, nil
}

// Create inserts a new calendar event
func (r *CalendarRepository) Create(ctx context.Context, event *models.CalendarEvent) error {
	recurrenceJSON, _ := json.Marshal(event.Recurrence)
	remindersJSON, _ := json.Marshal(event.Reminders)

	query := `
		INSERT INTO calendar_events (id, user_id, title, description, start_date, end_date, all_day,
		                              color, recurrence, reminders, linked_task_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		event.ID, event.UserID, event.Title, event.Description, event.StartDate, event.EndDate,
		event.AllDay, event.Color, recurrenceJSON, remindersJSON, event.LinkedTaskID,
		event.CreatedAt, event.UpdatedAt,
	)
	return err
}

// Update updates a calendar event
func (r *CalendarRepository) Update(ctx context.Context, event *models.CalendarEvent) error {
	recurrenceJSON, _ := json.Marshal(event.Recurrence)
	remindersJSON, _ := json.Marshal(event.Reminders)

	query := `
		UPDATE calendar_events SET
			title = $3, description = $4, start_date = $5, end_date = $6, all_day = $7,
			color = $8, recurrence = $9, reminders = $10, linked_task_id = $11, updated_at = $12
		WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query,
		event.ID, event.UserID, event.Title, event.Description, event.StartDate, event.EndDate,
		event.AllDay, event.Color, recurrenceJSON, remindersJSON, event.LinkedTaskID,
		event.UpdatedAt,
	)
	return err
}

// Delete deletes a calendar event
func (r *CalendarRepository) Delete(ctx context.Context, eventID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM calendar_events WHERE id = $1 AND user_id = $2", eventID, userID)
	return err
}

// DeleteByTaskID deletes all calendar events linked to a specific task
func (r *CalendarRepository) DeleteByTaskID(ctx context.Context, userID uuid.UUID, taskID string) (int64, error) {
	result, err := r.db.Exec(ctx,
		"DELETE FROM calendar_events WHERE user_id = $1 AND linked_task_id = $2",
		userID, taskID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
