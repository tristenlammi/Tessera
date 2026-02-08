package models

import (
	"time"

	"github.com/google/uuid"
)

// Activity represents an activity log entry
type Activity struct {
	ID           uuid.UUID              `json:"id"`
	UserID       uuid.UUID              `json:"userId"`
	User         *User                  `json:"user,omitempty"`
	Action       string                 `json:"action"`       // login, logout, upload, download, delete, share, etc.
	ResourceType string                 `json:"resourceType"` // file, folder, share, user
	ResourceID   string                 `json:"resourceId"`
	IPAddress    string                 `json:"ipAddress"`
	UserAgent    string                 `json:"userAgent"`
	Details      map[string]interface{} `json:"details"`
	CreatedAt    time.Time              `json:"createdAt"`
}
