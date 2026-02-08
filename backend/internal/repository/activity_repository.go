package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

type ActivityRepository struct {
	db *pgxpool.Pool
}

func NewActivityRepository(db *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Create creates a new activity log entry
func (r *ActivityRepository) Create(ctx context.Context, activity *models.Activity) error {
	query := `
		INSERT INTO activities (id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	activity.ID = uuid.New()
	activity.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		activity.ID,
		activity.UserID,
		activity.Action,
		activity.ResourceType,
		activity.ResourceID,
		activity.IPAddress,
		activity.UserAgent,
		activity.Details,
		activity.CreatedAt,
	)
	return err
}

// Log creates a new activity log entry with the given parameters
func (r *ActivityRepository) Log(ctx context.Context, userID uuid.UUID, action, resourceType, resourceID, ipAddress, userAgent string, details map[string]interface{}) error {
	activity := &models.Activity{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Details:      details,
	}
	return r.Create(ctx, activity)
}

// DeleteOlderThan deletes activities older than the specified number of days
func (r *ActivityRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := r.db.Exec(ctx, "DELETE FROM activities WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
