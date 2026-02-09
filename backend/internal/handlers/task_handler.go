package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
)

// TaskHandler handles task management endpoints
type TaskHandler struct {
	log      zerolog.Logger
	taskRepo *repository.TaskRepository
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(log zerolog.Logger, taskRepo *repository.TaskRepository) *TaskHandler {
	return &TaskHandler{
		log:      log,
		taskRepo: taskRepo,
	}
}

// ListTasks returns all tasks for a user
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	tasks, err := h.taskRepo.ListByUser(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list tasks")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tasks",
		})
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
	})
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Title              string               `json:"title"`
		Description        string               `json:"description"`
		Status             string               `json:"status"`
		Priority           string               `json:"priority"`
		DueDate            *time.Time           `json:"dueDate"`
		GroupID            *uuid.UUID           `json:"groupId"`
		Recurrence         *models.RecurrenceRule `json:"recurrence"`
		Checklist          []models.ChecklistItem `json:"checklist"`
		Tags               []string             `json:"tags"`
		LinkedEmailID      *string              `json:"linkedEmailId"`
		LinkedEmailSubject *string              `json:"linkedEmailSubject"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.log.Error().Err(err).Str("body", string(c.Body())).Msg("CreateTask: BodyParser failed")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title is required",
		})
	}

	if req.Status == "" {
		req.Status = "todo"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.Checklist == nil {
		req.Checklist = []models.ChecklistItem{}
	}

	// Assign IDs to checklist items if missing
	for i := range req.Checklist {
		if req.Checklist[i].ID == uuid.Nil {
			req.Checklist[i].ID = uuid.New()
		}
		req.Checklist[i].Order = i
	}

	// Determine order
	maxOrder, _ := h.taskRepo.GetMaxOrder(c.Context(), userID, req.Status)

	now := time.Now()
	task := &models.Task{
		ID:                 uuid.New(),
		UserID:             userID,
		Title:              req.Title,
		Description:        req.Description,
		Status:             req.Status,
		Priority:           req.Priority,
		DueDate:            req.DueDate,
		GroupID:            req.GroupID,
		Recurrence:         req.Recurrence,
		Checklist:          req.Checklist,
		Tags:               req.Tags,
		Order:              maxOrder + 1,
		LinkedEmailID:      req.LinkedEmailID,
		LinkedEmailSubject: req.LinkedEmailSubject,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.taskRepo.Create(c.Context(), task); err != nil {
		h.log.Error().Err(err).Msg("Failed to create task")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create task",
		})
	}

	h.log.Info().
		Str("task_id", task.ID.String()).
		Str("user_id", userID.String()).
		Msg("Task created")

	return c.Status(fiber.StatusCreated).JSON(task)
}

// GetTask returns a single task
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	task, err := h.taskRepo.GetByID(c.Context(), taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	return c.JSON(task)
}

// UpdateTask updates a task
func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	var req struct {
		Title              *string                 `json:"title"`
		Description        *string                 `json:"description"`
		Status             *string                 `json:"status"`
		Priority           *string                 `json:"priority"`
		DueDate            *time.Time              `json:"dueDate"`
		GroupID            *uuid.UUID              `json:"groupId"`
		Recurrence         *models.RecurrenceRule  `json:"recurrence"`
		Checklist          *[]models.ChecklistItem `json:"checklist"`
		Tags               []string                `json:"tags"`
		LinkedEmailID      *string                 `json:"linkedEmailId"`
		LinkedEmailSubject *string                 `json:"linkedEmailSubject"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	task, err := h.taskRepo.GetByID(c.Context(), taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		// Handle completion
		if *req.Status == "done" && task.Status != "done" {
			now := time.Now()
			task.CompletedAt = &now

			// Handle recurrence - create new task
			if task.Recurrence != nil {
				h.createRecurringTask(c, userID, *task)
			}
		}
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.GroupID != nil {
		task.GroupID = req.GroupID
	}
	if req.Recurrence != nil {
		task.Recurrence = req.Recurrence
	}
	if req.Checklist != nil {
		for j := range *req.Checklist {
			if (*req.Checklist)[j].ID == uuid.Nil {
				(*req.Checklist)[j].ID = uuid.New()
			}
			(*req.Checklist)[j].Order = j
		}
		task.Checklist = *req.Checklist
	}
	if req.Tags != nil {
		task.Tags = req.Tags
	}
	if req.LinkedEmailID != nil {
		task.LinkedEmailID = req.LinkedEmailID
	}
	if req.LinkedEmailSubject != nil {
		task.LinkedEmailSubject = req.LinkedEmailSubject
	}

	task.UpdatedAt = time.Now()

	if err := h.taskRepo.Update(c.Context(), task); err != nil {
		h.log.Error().Err(err).Msg("Failed to update task")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update task",
		})
	}

	return c.JSON(task)
}

// MoveTask moves a task to a new status/position
func (h *TaskHandler) MoveTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	var req struct {
		Status string `json:"status"`
		Order  int    `json:"order"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	task, err := h.taskRepo.GetByID(c.Context(), taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}

	oldStatus := task.Status
	task.Status = req.Status
	task.Order = req.Order
	task.UpdatedAt = time.Now()

	// Handle completion
	if req.Status == "done" && oldStatus != "done" {
		now := time.Now()
		task.CompletedAt = &now

		if task.Recurrence != nil {
			h.createRecurringTask(c, userID, *task)
		}
	}

	if err := h.taskRepo.Update(c.Context(), task); err != nil {
		h.log.Error().Err(err).Msg("Failed to move task")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to move task",
		})
	}

	return c.JSON(task)
}

// ReorderTasks reorders tasks in a column
func (h *TaskHandler) ReorderTasks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Status  string      `json:"status"`
		TaskIDs []uuid.UUID `json:"taskIds"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.taskRepo.Reorder(c.Context(), userID, req.Status, req.TaskIDs); err != nil {
		h.log.Error().Err(err).Msg("Failed to reorder tasks")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reorder tasks",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// DeleteTask deletes a task
func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid task ID",
		})
	}

	if err := h.taskRepo.Delete(c.Context(), taskID, userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete task")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete task",
		})
	}

	h.log.Info().
		Str("task_id", taskID.String()).
		Str("user_id", userID.String()).
		Msg("Task deleted")

	return c.SendStatus(fiber.StatusNoContent)
}

// createRecurringTask creates a new instance of a recurring task
func (h *TaskHandler) createRecurringTask(c *fiber.Ctx, userID uuid.UUID, completedTask models.Task) {
	if completedTask.Recurrence == nil || completedTask.DueDate == nil {
		return
	}

	newRecurrence := *completedTask.Recurrence
	newRecurrence.OccurrencesCompleted = completedTask.Recurrence.OccurrencesCompleted + 1

	if newRecurrence.Occurrences != nil && newRecurrence.OccurrencesCompleted >= *newRecurrence.Occurrences {
		h.log.Info().
			Str("task_id", completedTask.ID.String()).
			Int("occurrences", newRecurrence.OccurrencesCompleted).
			Msg("Recurring task reached occurrence limit, not creating new instance")
		return
	}

	nextDue := calculateNextDueDate(*completedTask.DueDate, newRecurrence)
	if nextDue == nil {
		return
	}

	maxOrder, _ := h.taskRepo.GetMaxOrder(c.Context(), userID, "todo")

	var newChecklist []models.ChecklistItem
	for _, item := range completedTask.Checklist {
		newChecklist = append(newChecklist, models.ChecklistItem{
			ID:        uuid.New(),
			Title:     item.Title,
			Completed: false,
			Order:     item.Order,
		})
	}

	now := time.Now()
	newTask := &models.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       completedTask.Title,
		Description: completedTask.Description,
		Status:      "todo",
		Priority:    completedTask.Priority,
		DueDate:     nextDue,
		GroupID:     completedTask.GroupID,
		Recurrence:  &newRecurrence,
		Checklist:   newChecklist,
		Tags:        completedTask.Tags,
		Order:       maxOrder + 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.taskRepo.Create(c.Context(), newTask); err != nil {
		h.log.Error().Err(err).Msg("Failed to create recurring task")
		return
	}

	h.log.Info().
		Str("task_id", newTask.ID.String()).
		Str("from_task", completedTask.ID.String()).
		Int("occurrence", newRecurrence.OccurrencesCompleted).
		Msg("Recurring task created")
}

// calculateNextDueDate calculates the next due date based on recurrence rule
func calculateNextDueDate(current time.Time, rule models.RecurrenceRule) *time.Time {
	var next time.Time

	switch rule.Type {
	case "daily":
		next = current.AddDate(0, 0, rule.Interval)

	case "weekly":
		if len(rule.DaysOfWeek) > 0 {
			next = current.AddDate(0, 0, 1)
			weeksPassed := 0
			for i := 0; i < 365; i++ {
				currentWeekday := int(next.Weekday())
				daysFromStart := int(next.Sub(current.AddDate(0, 0, 1)).Hours() / 24)
				currentWeekNum := daysFromStart / 7

				if currentWeekNum > weeksPassed && currentWeekNum%rule.Interval != 0 {
					daysToAdd := (rule.Interval - (currentWeekNum % rule.Interval)) * 7
					next = next.AddDate(0, 0, daysToAdd)
					weeksPassed = currentWeekNum + (rule.Interval - (currentWeekNum % rule.Interval))
					continue
				}

				for _, day := range rule.DaysOfWeek {
					if currentWeekday == day {
						goto foundDay
					}
				}
				next = next.AddDate(0, 0, 1)
			}
		foundDay:
		} else {
			next = current.AddDate(0, 0, rule.Interval*7)
		}

	case "monthly":
		if rule.DayOfMonth != nil {
			next = time.Date(
				current.Year(),
				current.Month()+time.Month(rule.Interval),
				*rule.DayOfMonth,
				current.Hour(),
				current.Minute(),
				current.Second(),
				0,
				current.Location(),
			)
			for next.Day() != *rule.DayOfMonth {
				next = next.AddDate(0, 0, -1)
			}
		} else {
			next = current.AddDate(0, rule.Interval, 0)
		}

	case "yearly":
		next = current.AddDate(rule.Interval, 0, 0)

	default:
		return nil
	}

	if rule.EndDate != nil && next.After(*rule.EndDate) {
		return nil
	}

	return &next
}

// --- Task Groups ---

// ListGroups returns all task groups for a user
func (h *TaskHandler) ListGroups(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	groups, err := h.taskRepo.ListGroups(c.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to list task groups")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch task groups",
		})
	}

	return c.JSON(fiber.Map{
		"groups": groups,
	})
}

// CreateGroup creates a new task group
func (h *TaskHandler) CreateGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Name       string               `json:"name"`
		Color      string               `json:"color"`
		Recurrence *models.RecurrenceRule `json:"recurrence"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	if req.Color == "" {
		req.Color = "#3b82f6"
	}

	group := &models.TaskGroup{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       req.Name,
		Color:      req.Color,
		Recurrence: req.Recurrence,
		CreatedAt:  time.Now(),
	}

	if err := h.taskRepo.CreateGroup(c.Context(), group); err != nil {
		h.log.Error().Err(err).Msg("Failed to create task group")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create task group",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(group)
}

// UpdateGroup updates a task group
func (h *TaskHandler) UpdateGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	groupID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	var req struct {
		Name       *string               `json:"name"`
		Color      *string               `json:"color"`
		Recurrence *models.RecurrenceRule `json:"recurrence"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	group, err := h.taskRepo.GetGroupByID(c.Context(), groupID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Group not found",
		})
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Color != nil {
		group.Color = *req.Color
	}
	if req.Recurrence != nil {
		group.Recurrence = req.Recurrence
	}

	if err := h.taskRepo.UpdateGroup(c.Context(), group); err != nil {
		h.log.Error().Err(err).Msg("Failed to update task group")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update task group",
		})
	}

	return c.JSON(group)
}

// DeleteGroup deletes a task group
func (h *TaskHandler) DeleteGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	groupID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	// Ungroup tasks first
	if err := h.taskRepo.UngroupByGroupID(c.Context(), userID, groupID); err != nil {
		h.log.Error().Err(err).Msg("Failed to ungroup tasks")
	}

	if err := h.taskRepo.DeleteGroup(c.Context(), groupID, userID); err != nil {
		h.log.Error().Err(err).Msg("Failed to delete task group")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete task group",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
