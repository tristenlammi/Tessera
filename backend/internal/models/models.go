package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	Timezone     string     `json:"timezone"`
	StorageUsed  int64      `json:"storage_used"`
	StorageLimit int64      `json:"storage_limit"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	// Two-Factor Authentication
	TOTPSecret  string   `json:"-"` // Encrypted TOTP secret
	TOTPEnabled bool     `json:"totp_enabled"`
	BackupCodes []string `json:"-"` // Hashed backup codes
}

// UserRole constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
	RoleOwner = "owner"
)

// Session represents an active user session
type Session struct {
	ID           string    `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	RefreshToken string    `json:"-"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// File represents a file or folder in the virtual file system
type File struct {
	ID         uuid.UUID  `json:"id"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"`
	OwnerID    uuid.UUID  `json:"owner_id"`
	Name       string     `json:"name"`
	IsFolder   bool       `json:"is_folder"`
	Size       int64      `json:"size"`
	MimeType   string     `json:"mime_type,omitempty"`
	StorageKey string     `json:"-"`
	Hash       string     `json:"-"`
	IsStarred  bool       `json:"is_starred"`
	IsTrashed  bool       `json:"is_trashed"`
	TrashedAt  *time.Time `json:"trashed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	AccessedAt *time.Time `json:"accessed_at,omitempty"`
}

// FileVersion represents a previous version of a file
type FileVersion struct {
	ID         uuid.UUID `json:"id"`
	FileID     uuid.UUID `json:"file_id"`
	Version    int       `json:"version"`
	Size       int64     `json:"size"`
	StorageKey string    `json:"-"`
	Hash       string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  uuid.UUID `json:"created_by"`
}

// Share represents a file or folder sharing configuration
type Share struct {
	ID             uuid.UUID  `json:"id"`
	FileID         uuid.UUID  `json:"file_id"`
	OwnerID        uuid.UUID  `json:"owner_id"`
	SharedWith     *uuid.UUID `json:"shared_with,omitempty"`
	PublicToken    *string    `json:"public_token,omitempty"`
	Permission     string     `json:"permission"`
	PasswordHash   *string    `json:"-"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	MaxDownloads   *int       `json:"max_downloads,omitempty"`
	DownloadCount  int        `json:"download_count"`
	ViewCount      int        `json:"view_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// SharedFile represents a file shared with a user (used in queries)
type SharedFile struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	IsFolder   bool      `json:"is_folder"`
	Size       int64     `json:"size"`
	MimeType   string    `json:"mime_type"`
	Permission string    `json:"permission"`
	OwnerID    uuid.UUID `json:"owner_id"`
	OwnerName  string    `json:"owner_name"`
	OwnerEmail string    `json:"owner_email"`
	SharedAt   time.Time `json:"shared_at"`
}

// AuditLog represents an immutable activity log entry
type AuditLog struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID uuid.UUID `json:"resource_id"`
	Details    string    `json:"details,omitempty"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuditAction constants
const (
	ActionCreate   = "create"
	ActionRead     = "read"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionDownload = "download"
	ActionShare    = "share"
	ActionLogin    = "login"
	ActionLogout   = "logout"
)
