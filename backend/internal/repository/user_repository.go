package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// UserRepository handles user database operations
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, storage_limit, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.StorageLimit,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetByID retrieves a user by their ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, timezone, storage_used, storage_limit, is_active, created_at, updated_at, last_login_at, totp_secret, totp_enabled, backup_codes
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	var totpSecret *string
	var backupCodes []string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.Timezone,
		&user.StorageUsed,
		&user.StorageLimit,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&totpSecret,
		&user.TOTPEnabled,
		&backupCodes,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if totpSecret != nil {
		user.TOTPSecret = *totpSecret
	}
	user.BackupCodes = backupCodes

	return user, err
}

// GetByEmail retrieves a user by their email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, timezone, storage_used, storage_limit, is_active, created_at, updated_at, last_login_at, totp_secret, totp_enabled, backup_codes
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	var totpSecret *string
	var backupCodes []string
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.Timezone,
		&user.StorageUsed,
		&user.StorageLimit,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&totpSecret,
		&user.TOTPEnabled,
		&backupCodes,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if totpSecret != nil {
		user.TOTPSecret = *totpSecret
	}
	user.BackupCodes = backupCodes

	return user, err
}

// Update modifies an existing user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, role = $4, storage_used = $5, is_active = $6, updated_at = $7, last_login_at = $8
		WHERE id = $1
	`

	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.Role,
		user.StorageUsed,
		user.IsActive,
		user.UpdatedAt,
		user.LastLoginAt,
	)

	return err
}

// UpdatePassword changes the user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, passwordHash, time.Now())
	return err
}

// UpdateTimezone updates the user's timezone preference
func (r *UserRepository) UpdateTimezone(ctx context.Context, userID uuid.UUID, timezone string) error {
	query := `
		UPDATE users
		SET timezone = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, timezone, time.Now())
	return err
}

// UpdateStorageUsed updates the user's storage usage
func (r *UserRepository) UpdateStorageUsed(ctx context.Context, userID uuid.UUID, delta int64) error {
	query := `
		UPDATE users
		SET storage_used = storage_used + $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, delta, time.Now())
	return err
}

// EmailExists checks if an email is already registered
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

// Delete removes a user from the database
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// UpdateTOTPSecret sets the TOTP secret for a user (during 2FA setup)
func (r *UserRepository) UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	query := `
		UPDATE users
		SET totp_secret = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, secret, time.Now())
	return err
}

// EnableTOTP enables 2FA for a user with the given backup codes
func (r *UserRepository) EnableTOTP(ctx context.Context, userID uuid.UUID, backupCodes []string) error {
	query := `
		UPDATE users
		SET totp_enabled = TRUE, backup_codes = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, backupCodes, time.Now())
	return err
}

// DisableTOTP disables 2FA for a user and clears related data
func (r *UserRepository) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET totp_enabled = FALSE, totp_secret = NULL, backup_codes = NULL, updated_at = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, time.Now())
	return err
}

// UpdateBackupCodes updates the backup codes for a user (after one is used)
func (r *UserRepository) UpdateBackupCodes(ctx context.Context, userID uuid.UUID, backupCodes []string) error {
	query := `
		UPDATE users
		SET backup_codes = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, backupCodes, time.Now())
	return err
}
