package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/middleware"
)

// TaskHandler handles task management endpoints
type TaskHandler struct {
	log zerolog.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(log zerolog.Logger) *TaskHandler {
	return &TaskHandler{
		log: log,
	}
}

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

// RecurrenceRule defines task recurrence
type RecurrenceRule struct {
	Type                 string     `json:"type"` // daily, weekly, monthly, yearly
	Interval             int        `json:"interval"`
	DaysOfWeek           []int      `json:"daysOfWeek,omitempty"`  // 0=Sunday, 1=Monday, ..., 6=Saturday
	DayOfMonth           *int       `json:"dayOfMonth,omitempty"`  // 1-31 for monthly
	EndDate              *time.Time `json:"endDate,omitempty"`     // End recurrence after this date
	Occurrences          *int       `json:"occurrences,omitempty"` // Max number of occurrences (total)
	OccurrencesCompleted int        `json:"occurrencesCompleted"`  // How many times completed
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

// In-memory storage (in production, use database)
var (
	tasks      = make(map[uuid.UUID][]Task)
	taskGroups = make(map[uuid.UUID][]TaskGroup)
)

// ListTasks returns all tasks for a user
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	userTasks, exists := tasks[userID]
	if !exists {
		userTasks = []Task{}
	}

	return c.JSON(fiber.Map{
		"tasks": userTasks,
	})
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Title              string          `json:"title"`
		Description        string          `json:"description"`
		Status             string          `json:"status"`
		Priority           string          `json:"priority"`
		DueDate            *time.Time      `json:"dueDate"`
		GroupID            *uuid.UUID      `json:"groupId"`
		Recurrence         *RecurrenceRule `json:"recurrence"`
		Checklist          []ChecklistItem `json:"checklist"`
		Tags               []string        `json:"tags"`
		LinkedEmailID      *string         `json:"linkedEmailId"`
		LinkedEmailSubject *string         `json:"linkedEmailSubject"`
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
		req.Checklist = []ChecklistItem{}
	}

	// Assign IDs to checklist items if missing
	for i := range req.Checklist {
		if req.Checklist[i].ID == uuid.Nil {
			req.Checklist[i].ID = uuid.New()
		}
		req.Checklist[i].Order = i
	}

	// Determine order
	userTasks := tasks[userID]
	maxOrder := 0
	for _, t := range userTasks {
		if t.Status == req.Status && t.Order > maxOrder {
			maxOrder = t.Order
		}
	}

	now := time.Now()
	task := Task{
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

	tasks[userID] = append(tasks[userID], task)

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

	userTasks := tasks[userID]
	for _, task := range userTasks {
		if task.ID == taskID {
			return c.JSON(task)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Task not found",
	})
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
		Title              *string          `json:"title"`
		Description        *string          `json:"description"`
		Status             *string          `json:"status"`
		Priority           *string          `json:"priority"`
		DueDate            *time.Time       `json:"dueDate"`
		GroupID            *uuid.UUID       `json:"groupId"`
		Recurrence         *RecurrenceRule  `json:"recurrence"`
		Checklist          *[]ChecklistItem `json:"checklist"`
		Tags               []string         `json:"tags"`
		LinkedEmailID      *string          `json:"linkedEmailId"`
		LinkedEmailSubject *string          `json:"linkedEmailSubject"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userTasks := tasks[userID]
	for i, task := range userTasks {
		if task.ID == taskID {
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
						h.createRecurringTask(userID, task)
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
				// Assign IDs to new checklist items
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
			tasks[userID][i] = task

			return c.JSON(task)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Task not found",
	})
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

	userTasks := tasks[userID]
	for i, task := range userTasks {
		if task.ID == taskID {
			oldStatus := task.Status
			task.Status = req.Status
			task.Order = req.Order
			task.UpdatedAt = time.Now()

			// Handle completion
			if req.Status == "done" && oldStatus != "done" {
				now := time.Now()
				task.CompletedAt = &now

				if task.Recurrence != nil {
					h.createRecurringTask(userID, task)
				}
			}

			tasks[userID][i] = task
			return c.JSON(task)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Task not found",
	})
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

	userTasks := tasks[userID]
	taskMap := make(map[uuid.UUID]*Task)
	for i := range userTasks {
		taskMap[userTasks[i].ID] = &userTasks[i]
	}

	for order, taskID := range req.TaskIDs {
		if task, exists := taskMap[taskID]; exists {
			task.Status = req.Status
			task.Order = order
			task.UpdatedAt = time.Now()
		}
	}

	tasks[userID] = userTasks

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

	userTasks := tasks[userID]
	for i, task := range userTasks {
		if task.ID == taskID {
			tasks[userID] = append(userTasks[:i], userTasks[i+1:]...)

			h.log.Info().
				Str("task_id", taskID.String()).
				Str("user_id", userID.String()).
				Msg("Task deleted")

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Task not found",
	})
}

// createRecurringTask creates a new instance of a recurring task
func (h *TaskHandler) createRecurringTask(userID uuid.UUID, completedTask Task) {
	if completedTask.Recurrence == nil || completedTask.DueDate == nil {
		return
	}

	// Copy recurrence rule to increment occurrencesCompleted
	newRecurrence := *completedTask.Recurrence
	newRecurrence.OccurrencesCompleted = completedTask.Recurrence.OccurrencesCompleted + 1

	// Check occurrence limit
	if newRecurrence.Occurrences != nil && newRecurrence.OccurrencesCompleted >= *newRecurrence.Occurrences {
		h.log.Info().
			Str("task_id", completedTask.ID.String()).
			Int("occurrences", newRecurrence.OccurrencesCompleted).
			Msg("Recurring task reached occurrence limit, not creating new instance")
		return
	}

	nextDue := h.calculateNextDueDate(*completedTask.DueDate, newRecurrence)
	if nextDue == nil {
		return
	}

	// Determine order
	userTasks := tasks[userID]
	maxOrder := 0
	for _, t := range userTasks {
		if t.Status == "todo" && t.Order > maxOrder {
			maxOrder = t.Order
		}
	}

	// Reset checklist items to unchecked for new recurrence
	var newChecklist []ChecklistItem
	for _, item := range completedTask.Checklist {
		newChecklist = append(newChecklist, ChecklistItem{
			ID:        uuid.New(),
			Title:     item.Title,
			Completed: false,
			Order:     item.Order,
		})
	}

	now := time.Now()
	newTask := Task{
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

	tasks[userID] = append(tasks[userID], newTask)

	h.log.Info().
		Str("task_id", newTask.ID.String()).
		Str("from_task", completedTask.ID.String()).
		Int("occurrence", newRecurrence.OccurrencesCompleted).
		Msg("Recurring task created")
}

// calculateNextDueDate calculates the next due date based on recurrence rule
func (h *TaskHandler) calculateNextDueDate(current time.Time, rule RecurrenceRule) *time.Time {
	var next time.Time

	switch rule.Type {
	case "daily":
		next = current.AddDate(0, 0, rule.Interval)

	case "weekly":
		if len(rule.DaysOfWeek) > 0 {
			// Find next occurrence on one of the selected days
			next = current.AddDate(0, 0, 1) // Start from tomorrow
			weeksPassed := 0
			for i := 0; i < 365; i++ { // Safety limit
				currentWeekday := int(next.Weekday())
				daysFromStart := int(next.Sub(current.AddDate(0, 0, 1)).Hours() / 24)
				currentWeekNum := daysFromStart / 7

				// Check if we've moved to a new week interval
				if currentWeekNum > weeksPassed && currentWeekNum%rule.Interval != 0 {
					// Skip to next valid week
					daysToAdd := (rule.Interval - (currentWeekNum % rule.Interval)) * 7
					next = next.AddDate(0, 0, daysToAdd)
					weeksPassed = currentWeekNum + (rule.Interval - (currentWeekNum % rule.Interval))
					continue
				}

				// Check if current day is in daysOfWeek
				for _, day := range rule.DaysOfWeek {
					if currentWeekday == day {
						goto foundDay
					}
				}
				next = next.AddDate(0, 0, 1)
			}
		foundDay:
		} else {
			// Simple weekly: just add N weeks
			next = current.AddDate(0, 0, rule.Interval*7)
		}

	case "monthly":
		if rule.DayOfMonth != nil {
			// Specific day of month
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
			// Handle cases where day doesn't exist in month (e.g., Feb 31)
			for next.Day() != *rule.DayOfMonth {
				next = next.AddDate(0, 0, -1)
			}
		} else {
			// Same day of month
			next = current.AddDate(0, rule.Interval, 0)
		}

	case "yearly":
		next = current.AddDate(rule.Interval, 0, 0)

	default:
		return nil
	}

	// Check end date
	if rule.EndDate != nil && next.After(*rule.EndDate) {
		return nil
	}

	return &next
}

// --- Task Groups ---

// ListGroups returns all task groups for a user
func (h *TaskHandler) ListGroups(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	userGroups, exists := taskGroups[userID]
	if !exists {
		userGroups = []TaskGroup{}
	}

	return c.JSON(fiber.Map{
		"groups": userGroups,
	})
}

// CreateGroup creates a new task group
func (h *TaskHandler) CreateGroup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Name       string          `json:"name"`
		Color      string          `json:"color"`
		Recurrence *RecurrenceRule `json:"recurrence"`
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

	group := TaskGroup{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       req.Name,
		Color:      req.Color,
		Recurrence: req.Recurrence,
		CreatedAt:  time.Now(),
	}

	taskGroups[userID] = append(taskGroups[userID], group)

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
		Name       *string         `json:"name"`
		Color      *string         `json:"color"`
		Recurrence *RecurrenceRule `json:"recurrence"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userGroups := taskGroups[userID]
	for i, group := range userGroups {
		if group.ID == groupID {
			if req.Name != nil {
				group.Name = *req.Name
			}
			if req.Color != nil {
				group.Color = *req.Color
			}
			if req.Recurrence != nil {
				group.Recurrence = req.Recurrence
			}

			taskGroups[userID][i] = group
			return c.JSON(group)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Group not found",
	})
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

	userGroups := taskGroups[userID]
	for i, group := range userGroups {
		if group.ID == groupID {
			taskGroups[userID] = append(userGroups[:i], userGroups[i+1:]...)

			// Ungroup tasks
			userTasks := tasks[userID]
			for j := range userTasks {
				if userTasks[j].GroupID != nil && *userTasks[j].GroupID == groupID {
					userTasks[j].GroupID = nil
				}
			}

			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Group not found",
	})
}
