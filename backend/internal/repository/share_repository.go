package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tessera/tessera/internal/models"
)

type ShareRepository struct {
	db *pgxpool.Pool
}

func NewShareRepository(db *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{db: db}
}

// Create creates a new share
func (r *ShareRepository) Create(ctx context.Context, share *models.Share) error {
	query := `
		INSERT INTO shares (id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query,
		share.ID,
		share.FileID,
		share.OwnerID,
		share.SharedWith,
		share.PublicToken,
		share.Permission,
		share.PasswordHash,
		share.ExpiresAt,
		share.MaxDownloads,
		share.DownloadCount,
		share.CreatedAt,
	)
	return err
}

// FindByID finds a share by ID
func (r *ShareRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, created_at
		FROM shares
		WHERE id = $1
	`
	var share models.Share
	err := r.db.QueryRow(ctx, query, id).Scan(
		&share.ID,
		&share.FileID,
		&share.OwnerID,
		&share.SharedWith,
		&share.PublicToken,
		&share.Permission,
		&share.PasswordHash,
		&share.ExpiresAt,
		&share.MaxDownloads,
		&share.DownloadCount,
		&share.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// FindByToken finds a share by public token
func (r *ShareRepository) FindByToken(ctx context.Context, token string) (*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, created_at
		FROM shares
		WHERE public_token = $1
	`
	var share models.Share
	err := r.db.QueryRow(ctx, query, token).Scan(
		&share.ID,
		&share.FileID,
		&share.OwnerID,
		&share.SharedWith,
		&share.PublicToken,
		&share.Permission,
		&share.PasswordHash,
		&share.ExpiresAt,
		&share.MaxDownloads,
		&share.DownloadCount,
		&share.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &share, nil
}

// FindByFile returns all shares for a file
func (r *ShareRepository) FindByFile(ctx context.Context, fileID uuid.UUID) ([]models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, created_at
		FROM shares
		WHERE file_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []models.Share
	for rows.Next() {
		var share models.Share
		if err := rows.Scan(
			&share.ID,
			&share.FileID,
			&share.OwnerID,
			&share.SharedWith,
			&share.PublicToken,
			&share.Permission,
			&share.PasswordHash,
			&share.ExpiresAt,
			&share.MaxDownloads,
			&share.DownloadCount,
			&share.CreatedAt,
		); err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, nil
}

// Delete removes a share
func (r *ShareRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM shares WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Update updates a share
func (r *ShareRepository) Update(ctx context.Context, share *models.Share) error {
	query := `
		UPDATE shares
		SET permission = $2, password_hash = $3, expires_at = $4, max_downloads = $5, download_count = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		share.ID,
		share.Permission,
		share.PasswordHash,
		share.ExpiresAt,
		share.MaxDownloads,
		share.DownloadCount,
	)
	return err
}

// IncrementDownloadCount increments the download count for a share
func (r *ShareRepository) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE shares SET download_count = download_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Count returns total number of shares
func (r *ShareRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM shares").Scan(&count)
	return count, err
}
