package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

// TaskRepository handles task database operations
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// ListByUser returns all tasks for a user
func (r *TaskRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Task, error) {
	query := `
		SELECT id, user_id, group_id, title, description, status, priority, due_date,
		       recurrence, checklist, tags, sort_order, linked_email_id, linked_email_subject,
		       completed_at, created_at, updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var recurrenceJSON, checklistJSON []byte
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.GroupID, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.DueDate, &recurrenceJSON, &checklistJSON,
			&t.Tags, &t.Order, &t.LinkedEmailID, &t.LinkedEmailSubject,
			&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if recurrenceJSON != nil {
			var rec models.RecurrenceRule
			if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
				t.Recurrence = &rec
			}
		}
		if checklistJSON != nil {
			json.Unmarshal(checklistJSON, &t.Checklist)
		}
		if t.Checklist == nil {
			t.Checklist = []models.ChecklistItem{}
		}
		if t.Tags == nil {
			t.Tags = []string{}
		}
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}
	return tasks, nil
}

// GetByID returns a single task by ID, scoped to a user
func (r *TaskRepository) GetByID(ctx context.Context, taskID, userID uuid.UUID) (*models.Task, error) {
	query := `
		SELECT id, user_id, group_id, title, description, status, priority, due_date,
		       recurrence, checklist, tags, sort_order, linked_email_id, linked_email_subject,
		       completed_at, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND user_id = $2
	`

	var t models.Task
	var recurrenceJSON, checklistJSON []byte
	err := r.db.QueryRow(ctx, query, taskID, userID).Scan(
		&t.ID, &t.UserID, &t.GroupID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.DueDate, &recurrenceJSON, &checklistJSON,
		&t.Tags, &t.Order, &t.LinkedEmailID, &t.LinkedEmailSubject,
		&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if recurrenceJSON != nil {
		var rec models.RecurrenceRule
		if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
			t.Recurrence = &rec
		}
	}
	if checklistJSON != nil {
		json.Unmarshal(checklistJSON, &t.Checklist)
	}
	if t.Checklist == nil {
		t.Checklist = []models.ChecklistItem{}
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return &t, nil
}

// Create inserts a new task
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	recurrenceJSON, _ := json.Marshal(task.Recurrence)
	checklistJSON, _ := json.Marshal(task.Checklist)

	query := `
		INSERT INTO tasks (id, user_id, group_id, title, description, status, priority, due_date,
		                    recurrence, checklist, tags, sort_order, linked_email_id, linked_email_subject,
		                    completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.Exec(ctx, query,
		task.ID, task.UserID, task.GroupID, task.Title, task.Description,
		task.Status, task.Priority, task.DueDate, recurrenceJSON, checklistJSON,
		task.Tags, task.Order, task.LinkedEmailID, task.LinkedEmailSubject,
		task.CompletedAt, task.CreatedAt, task.UpdatedAt,
	)
	return err
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	recurrenceJSON, _ := json.Marshal(task.Recurrence)
	checklistJSON, _ := json.Marshal(task.Checklist)

	query := `
		UPDATE tasks SET
			group_id = $3, title = $4, description = $5, status = $6, priority = $7,
			due_date = $8, recurrence = $9, checklist = $10, tags = $11, sort_order = $12,
			linked_email_id = $13, linked_email_subject = $14, completed_at = $15, updated_at = $16
		WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query,
		task.ID, task.UserID, task.GroupID, task.Title, task.Description,
		task.Status, task.Priority, task.DueDate, recurrenceJSON, checklistJSON,
		task.Tags, task.Order, task.LinkedEmailID, task.LinkedEmailSubject,
		task.CompletedAt, task.UpdatedAt,
	)
	return err
}

// Delete deletes a task
func (r *TaskRepository) Delete(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE id = $1 AND user_id = $2", taskID, userID)
	return err
}

// GetMaxOrder returns the maximum sort order for tasks in a given status for a user
func (r *TaskRepository) GetMaxOrder(ctx context.Context, userID uuid.UUID, status string) (int, error) {
	var maxOrder int
	err := r.db.QueryRow(ctx,
		"SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE user_id = $1 AND status = $2",
		userID, status,
	).Scan(&maxOrder)
	return maxOrder, err
}

// Reorder updates the order of multiple tasks in a status column
func (r *TaskRepository) Reorder(ctx context.Context, userID uuid.UUID, status string, taskIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	for order, taskID := range taskIDs {
		_, err := tx.Exec(ctx,
			"UPDATE tasks SET status = $3, sort_order = $4, updated_at = $5 WHERE id = $1 AND user_id = $2",
			taskID, userID, status, order, now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// UngroupByGroupID sets group_id to NULL for all tasks in a given group
func (r *TaskRepository) UngroupByGroupID(ctx context.Context, userID, groupID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE tasks SET group_id = NULL, updated_at = NOW() WHERE user_id = $1 AND group_id = $2",
		userID, groupID,
	)
	return err
}

// --- Task Groups ---

// ListGroups returns all task groups for a user
func (r *TaskRepository) ListGroups(ctx context.Context, userID uuid.UUID) ([]models.TaskGroup, error) {
	query := `
		SELECT id, user_id, name, color, recurrence, created_at
		FROM task_groups
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.TaskGroup
	for rows.Next() {
		var g models.TaskGroup
		var recurrenceJSON []byte
		if err := rows.Scan(&g.ID, &g.UserID, &g.Name, &g.Color, &recurrenceJSON, &g.CreatedAt); err != nil {
			return nil, err
		}
		if recurrenceJSON != nil {
			var rec models.RecurrenceRule
			if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
				g.Recurrence = &rec
			}
		}
		groups = append(groups, g)
	}

	if groups == nil {
		groups = []models.TaskGroup{}
	}
	return groups, nil
}

// CreateGroup inserts a new task group
func (r *TaskRepository) CreateGroup(ctx context.Context, group *models.TaskGroup) error {
	recurrenceJSON, _ := json.Marshal(group.Recurrence)

	query := `
		INSERT INTO task_groups (id, user_id, name, color, recurrence, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		group.ID, group.UserID, group.Name, group.Color, recurrenceJSON, group.CreatedAt,
	)
	return err
}

// UpdateGroup updates a task group
func (r *TaskRepository) UpdateGroup(ctx context.Context, group *models.TaskGroup) error {
	recurrenceJSON, _ := json.Marshal(group.Recurrence)

	query := `
		UPDATE task_groups SET name = $3, color = $4, recurrence = $5
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.Exec(ctx, query,
		group.ID, group.UserID, group.Name, group.Color, recurrenceJSON,
	)
	return err
}

// DeleteGroup deletes a task group
func (r *TaskRepository) DeleteGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM task_groups WHERE id = $1 AND user_id = $2", groupID, userID)
	return err
}

// GetGroupByID returns a single task group
func (r *TaskRepository) GetGroupByID(ctx context.Context, groupID, userID uuid.UUID) (*models.TaskGroup, error) {
	var g models.TaskGroup
	var recurrenceJSON []byte
	err := r.db.QueryRow(ctx,
		"SELECT id, user_id, name, color, recurrence, created_at FROM task_groups WHERE id = $1 AND user_id = $2",
		groupID, userID,
	).Scan(&g.ID, &g.UserID, &g.Name, &g.Color, &recurrenceJSON, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	if recurrenceJSON != nil {
		var rec models.RecurrenceRule
		if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
			g.Recurrence = &rec
		}
	}
	return &g, nil
}
