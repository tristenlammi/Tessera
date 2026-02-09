package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
)

type CalendarHandler struct {
	log          zerolog.Logger
	calendarRepo *repository.CalendarRepository
}

func NewCalendarHandler(log zerolog.Logger, calendarRepo *repository.CalendarRepository) *CalendarHandler {
	return &CalendarHandler{
		log:          log,
		calendarRepo: calendarRepo,
	}
}

// ListEvents returns all events for a user within a date range
func (h *CalendarHandler) ListEvents(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	startStr := c.Query("start")
	endStr := c.Query("end")

	var startDate, endDate time.Time
	var err error

	if startStr != "" {
		startDate, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			startDate = time.Now().AddDate(0, -1, 0)
		}
	} else {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	if endStr != "" {
		endDate, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			endDate = time.Now().AddDate(0, 2, 0)
		}
	} else {
		endDate = time.Now().AddDate(0, 2, 0)
	}

	events, err := h.calendarRepo.ListByUser(c.Context(), userID, startDate, endDate)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list calendar events")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch events"})
	}

	return c.JSON(events)
}

// CreateEvent creates a new calendar event
func (h *CalendarHandler) CreateEvent(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var input struct {
		Title        string               `json:"title"`
		Description  string               `json:"description"`
		StartDate    string               `json:"startDate"`
		EndDate      string               `json:"endDate"`
		AllDay       bool                 `json:"allDay"`
		Color        string               `json:"color"`
		Recurrence   *models.RecurrenceRule `json:"recurrence"`
		LinkedTaskID *string              `json:"linkedTaskId"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	startDate, err := time.Parse(time.RFC3339, input.StartDate)
	if err != nil {
		startDate, err = time.Parse("2006-01-02T15:04:05", input.StartDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date format"})
		}
	}

	endDate, err := time.Parse(time.RFC3339, input.EndDate)
	if err != nil {
		endDate, err = time.Parse("2006-01-02T15:04:05", input.EndDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date format"})
		}
	}

	if input.Color == "" {
		input.Color = "#3b82f6"
	}

	now := time.Now()
	event := &models.CalendarEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Title:        input.Title,
		Description:  input.Description,
		StartDate:    startDate,
		EndDate:      endDate,
		AllDay:       input.AllDay,
		Color:        input.Color,
		Recurrence:   input.Recurrence,
		Reminders:    []models.EventReminder{},
		LinkedTaskID: input.LinkedTaskID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.calendarRepo.Create(c.Context(), event); err != nil {
		h.log.Error().Err(err).Msg("Failed to create calendar event")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create event"})
	}

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

	event, err := h.calendarRepo.GetByID(c.Context(), eventID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	return c.JSON(event)
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
		Title       string               `json:"title"`
		Description string               `json:"description"`
		StartDate   string               `json:"startDate"`
		EndDate     string               `json:"endDate"`
		AllDay      bool                 `json:"allDay"`
		Color       string               `json:"color"`
		Recurrence  *models.RecurrenceRule `json:"recurrence"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	event, err := h.calendarRepo.GetByID(c.Context(), eventID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Event not found"})
	}

	if input.Title != "" {
		event.Title = input.Title
	}
	event.Description = input.Description
	event.AllDay = input.AllDay
	if input.Color != "" {
		event.Color = input.Color
	}
	event.Recurrence = input.Recurrence

	if input.StartDate != "" {
		startDate, err := time.Parse(time.RFC3339, input.StartDate)
		if err != nil {
			startDate, _ = time.Parse("2006-01-02T15:04:05", input.StartDate)
		}
		event.StartDate = startDate
	}

	if input.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, input.EndDate)
		if err != nil {
			endDate, _ = time.Parse("2006-01-02T15:04:05", input.EndDate)
		}
		event.EndDate = endDate
	}

	event.UpdatedAt = time.Now()

	if err := h.calendarRepo.Update(c.Context(), event); err != nil {
		h.log.Error().Err(err).Msg("Failed to update calendar event")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update event"})
	}

	h.log.Info().
		Str("event_id", eventID.String()).
		Msg("Calendar event updated")

	return c.JSON(event)
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

	if err := h.calendarRepo.Delete(c.Context(), eventID, userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete calendar event")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete event"})
	}

	h.log.Info().
		Str("event_id", eventID.String()).
		Msg("Calendar event deleted")

	return c.SendStatus(fiber.StatusNoContent)
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

	deleted, err := h.calendarRepo.DeleteByTaskID(c.Context(), userID, taskID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to delete calendar events by task")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete events"})
	}

	h.log.Info().
		Str("task_id", taskID).
		Int64("deleted", deleted).
		Msg("Deleted calendar events linked to task")

	return c.JSON(fiber.Map{"deleted": deleted})
}
