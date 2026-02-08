package security

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEventType represents types of audit events
type AuditEventType string

const (
	// Authentication events
	AuditLogin          AuditEventType = "auth.login"
	AuditLogout         AuditEventType = "auth.logout"
	AuditLoginFailed    AuditEventType = "auth.login_failed"
	AuditPasswordChange AuditEventType = "auth.password_change"
	AuditPasswordReset  AuditEventType = "auth.password_reset"

	// File events
	AuditFileCreate   AuditEventType = "file.create"
	AuditFileUpdate   AuditEventType = "file.update"
	AuditFileDelete   AuditEventType = "file.delete"
	AuditFileDownload AuditEventType = "file.download"
	AuditFileMove     AuditEventType = "file.move"
	AuditFileCopy     AuditEventType = "file.copy"
	AuditFileRestore  AuditEventType = "file.restore"

	// Folder events
	AuditFolderCreate AuditEventType = "folder.create"
	AuditFolderDelete AuditEventType = "folder.delete"
	AuditFolderMove   AuditEventType = "folder.move"

	// Sharing events
	AuditShareCreate AuditEventType = "share.create"
	AuditShareRevoke AuditEventType = "share.revoke"
	AuditShareAccess AuditEventType = "share.access"

	// Admin events
	AuditUserCreate AuditEventType = "admin.user_create"
	AuditUserUpdate AuditEventType = "admin.user_update"
	AuditUserDelete AuditEventType = "admin.user_delete"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id,omitempty"`
	Type       AuditEventType         `json:"type"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Status     string                 `json:"status"` // success, failure
	Error      string                 `json:"error,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AuditLogger handles audit logging
type AuditLogger struct {
	db *pgxpool.Pool
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log records an audit event
func (a *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = a.db.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, event_type, action, resource, resource_id, details, ip_address, user_agent, status, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, event.ID, event.UserID, event.Type, event.Action, event.Resource, event.ResourceID, detailsJSON, event.IPAddress, event.UserAgent, event.Status, event.Error, event.CreatedAt)

	return err
}

// LogSuccess logs a successful event
func (a *AuditLogger) LogSuccess(ctx context.Context, userID string, eventType AuditEventType, action string, resource string, resourceID string, details map[string]interface{}, ipAddress string, userAgent string) error {
	return a.Log(ctx, &AuditEvent{
		UserID:     userID,
		Type:       eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "success",
	})
}

// LogFailure logs a failed event
func (a *AuditLogger) LogFailure(ctx context.Context, userID string, eventType AuditEventType, action string, resource string, resourceID string, errMsg string, ipAddress string, userAgent string) error {
	return a.Log(ctx, &AuditEvent{
		UserID:     userID,
		Type:       eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Status:     "failure",
		Error:      errMsg,
	})
}

// GetUserLogs retrieves audit logs for a user
func (a *AuditLogger) GetUserLogs(ctx context.Context, userID string, limit int, offset int) ([]*AuditEvent, error) {
	rows, err := a.db.Query(ctx, `
		SELECT id, user_id, event_type, action, resource, resource_id, details, ip_address, user_agent, status, error, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var detailsJSON []byte
		err := rows.Scan(&event.ID, &event.UserID, &event.Type, &event.Action, &event.Resource, &event.ResourceID, &detailsJSON, &event.IPAddress, &event.UserAgent, &event.Status, &event.Error, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &event.Details)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetResourceLogs retrieves audit logs for a specific resource
func (a *AuditLogger) GetResourceLogs(ctx context.Context, resourceID string, limit int) ([]*AuditEvent, error) {
	rows, err := a.db.Query(ctx, `
		SELECT id, user_id, event_type, action, resource, resource_id, details, ip_address, user_agent, status, error, created_at
		FROM audit_logs
		WHERE resource_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, resourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var detailsJSON []byte
		err := rows.Scan(&event.ID, &event.UserID, &event.Type, &event.Action, &event.Resource, &event.ResourceID, &detailsJSON, &event.IPAddress, &event.UserAgent, &event.Status, &event.Error, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &event.Details)
		}
		events = append(events, event)
	}

	return events, nil
}

// SearchLogs searches audit logs with filters
func (a *AuditLogger) SearchLogs(ctx context.Context, filters map[string]interface{}, limit int, offset int) ([]*AuditEvent, int, error) {
	query := `SELECT id, user_id, event_type, action, resource, resource_id, details, ip_address, user_agent, status, error, created_at FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	var args []interface{}
	argIndex := 1

	if userID, ok := filters["user_id"].(string); ok && userID != "" {
		query += " AND user_id = $" + string(rune('0'+argIndex))
		countQuery += " AND user_id = $" + string(rune('0'+argIndex))
		args = append(args, userID)
		argIndex++
	}

	if eventType, ok := filters["event_type"].(string); ok && eventType != "" {
		query += " AND event_type = $" + string(rune('0'+argIndex))
		countQuery += " AND event_type = $" + string(rune('0'+argIndex))
		args = append(args, eventType)
		argIndex++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = $" + string(rune('0'+argIndex))
		countQuery += " AND status = $" + string(rune('0'+argIndex))
		args = append(args, status)
		argIndex++
	}

	// Get total count
	var total int
	err := a.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT $" + string(rune('0'+argIndex)) + " OFFSET $" + string(rune('0'+argIndex+1))
	args = append(args, limit, offset)

	rows, err := a.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		var detailsJSON []byte
		err := rows.Scan(&event.ID, &event.UserID, &event.Type, &event.Action, &event.Resource, &event.ResourceID, &detailsJSON, &event.IPAddress, &event.UserAgent, &event.Status, &event.Error, &event.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &event.Details)
		}
		events = append(events, event)
	}

	return events, total, nil
}
