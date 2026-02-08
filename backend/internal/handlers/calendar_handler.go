package handlers

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type CalendarHandler struct {
	log zerolog.Logger
}

func NewCalendarHandler(log zerolog.Logger) *CalendarHandler {
	return &CalendarHandler{log: log}
}

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
	Minutes int       `json:"minutes"` // Minutes before event
}

// In-memory storage for calendar events (per user)
var (
	calendarEvents = make(map[uuid.UUID][]CalendarEvent)
	calendarMu     sync.RWMutex
)

// RegisterRoutes registers calendar routes
func (h *CalendarHandler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	calendar := app.Group("/api/calendar", authMiddleware)

	calendar.Get("/events", h.ListEvents)
	calendar.Post("/events", h.CreateEvent)
	calendar.Get("/events/:id", h.GetEvent)
	calendar.Put("/events/:id", h.UpdateEvent)
	calendar.Delete("/events/:id", h.DeleteEvent)
}

// ListEvents returns all events for a user within a date range
func (h *CalendarHandler) ListEvents(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse optional date range filters
	startStr := c.Query("start")
	endStr := c.Query("end")

	var startDate, endDate time.Time
	var err error

	if startStr != "" {
		startDate, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			startDate = time.Now().AddDate(0, -1, 0) // Default: 1 month ago
		}
	} else {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	if endStr != "" {
		endDate, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			endDate = time.Now().AddDate(0, 2, 0) // Default: 2 months from now
		}
	} else {
		endDate = time.Now().AddDate(0, 2, 0)
	}

	calendarMu.RLock()
	userEvents := calendarEvents[userID]
	calendarMu.RUnlock()

	// Filter events within date range
	filteredEvents := make([]CalendarEvent, 0)
	for _, event := range userEvents {
		// Event overlaps with range if event.start <= range.end AND event.end >= range.start
		if !event.StartDate.After(endDate) && !event.EndDate.Before(startDate) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return c.JSON(filteredEvents)
}

// CreateEvent creates a new calendar event
func (h *CalendarHandler) CreateEvent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var input struct {
		Title        string          `json:"title"`
		Description  string          `json:"description"`
		StartDate    string          `json:"startDate"`
		EndDate      string          `json:"endDate"`
		AllDay       bool            `json:"allDay"`
		Color        string          `json:"color"`
		Recurrence   *RecurrenceRule `json:"recurrence"`
		LinkedTaskID *string         `json:"linkedTaskId"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	startDate, err := time.Parse(time.RFC3339, input.StartDate)
	if err != nil {
		// Try parsing without timezone
		startDate, err = time.Parse("2006-01-02T15:04:05", input.StartDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date format"})
		}
	}

	endDate, err := time.Parse(time.RFC3339, input.EndDate)
	if err != nil {
		// Try parsing without timezone
		endDate, err = time.Parse("2006-01-02T15:04:05", input.EndDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date format"})
		}
	}

	if input.Color == "" {
		input.Color = "#3b82f6" // Default blue
	}

	now := time.Now()
	event := CalendarEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Title:        input.Title,
		Description:  input.Description,
		StartDate:    startDate,
		EndDate:      endDate,
		AllDay:       input.AllDay,
		Color:        input.Color,
		Recurrence:   input.Recurrence,
		Reminders:    []EventReminder{},
		LinkedTaskID: input.LinkedTaskID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	calendarMu.Lock()
	calendarEvents[userID] = append(calendarEvents[userID], event)
	calendarMu.Unlock()

	h.log.Info().
		Str("event_id", event.ID.String()).
		Str("title", event.Title).
		Msg("Calendar event created")

	return c.Status(fiber.StatusCreated).JSON(event)
}

// GetEvent returns a specific event
func (h *CalendarHandler) GetEvent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	calendarMu.RLock()
	userEvents := calendarEvents[userID]
	calendarMu.RUnlock()

	for _, event := range userEvents {
		if event.ID == eventID {
			return c.JSON(event)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
}

// UpdateEvent updates an existing event
func (h *CalendarHandler) UpdateEvent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	var input struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		StartDate   string          `json:"startDate"`
		EndDate     string          `json:"endDate"`
		AllDay      bool            `json:"allDay"`
		Color       string          `json:"color"`
		Recurrence  *RecurrenceRule `json:"recurrence"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	calendarMu.Lock()
	defer calendarMu.Unlock()

	userEvents := calendarEvents[userID]
	for i, event := range userEvents {
		if event.ID == eventID {
			if input.Title != "" {
				userEvents[i].Title = input.Title
			}
			userEvents[i].Description = input.Description
			userEvents[i].AllDay = input.AllDay
			if input.Color != "" {
				userEvents[i].Color = input.Color
			}
			userEvents[i].Recurrence = input.Recurrence

			if input.StartDate != "" {
				startDate, err := time.Parse(time.RFC3339, input.StartDate)
				if err != nil {
					startDate, _ = time.Parse("2006-01-02T15:04:05", input.StartDate)
				}
				userEvents[i].StartDate = startDate
			}

			if input.EndDate != "" {
				endDate, err := time.Parse(time.RFC3339, input.EndDate)
				if err != nil {
					endDate, _ = time.Parse("2006-01-02T15:04:05", input.EndDate)
				}
				userEvents[i].EndDate = endDate
			}

			userEvents[i].UpdatedAt = time.Now()
			calendarEvents[userID] = userEvents

			h.log.Info().
				Str("event_id", eventID.String()).
				Msg("Calendar event updated")

			return c.JSON(userEvents[i])
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
}

// DeleteEvent deletes an event
func (h *CalendarHandler) DeleteEvent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event ID"})
	}

	calendarMu.Lock()
	defer calendarMu.Unlock()

	userEvents := calendarEvents[userID]
	for i, event := range userEvents {
		if event.ID == eventID {
			calendarEvents[userID] = append(userEvents[:i], userEvents[i+1:]...)

			h.log.Info().
				Str("event_id", eventID.String()).
				Msg("Calendar event deleted")

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
}

// DeleteEventByTask deletes all calendar events linked to a specific task
func (h *CalendarHandler) DeleteEventByTask(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	taskID := c.Params("taskId")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Task ID is required"})
	}

	calendarMu.Lock()
	defer calendarMu.Unlock()

	userEvents := calendarEvents[userID]
	filtered := make([]CalendarEvent, 0, len(userEvents))
	deleted := 0
	for _, event := range userEvents {
		if event.LinkedTaskID != nil && *event.LinkedTaskID == taskID {
			deleted++
			continue
		}
		filtered = append(filtered, event)
	}
	calendarEvents[userID] = filtered

	h.log.Info().
		Str("task_id", taskID).
		Int("deleted", deleted).
		Msg("Deleted calendar events linked to task")

	return c.JSON(fiber.Map{"deleted": deleted})
}
