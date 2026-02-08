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
	ErrFileNotFound = errors.New("file not found")
)

// FileRepository handles file database operations
type FileRepository struct {
	db *pgxpool.Pool
}

// NewFileRepository creates a new file repository
func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

// Create inserts a new file or folder into the database
func (r *FileRepository) Create(ctx context.Context, file *models.File) error {
	query := `
		INSERT INTO files (id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	file.ID = uuid.New()
	file.CreatedAt = time.Now()
	file.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		file.ID,
		file.ParentID,
		file.OwnerID,
		file.Name,
		file.IsFolder,
		file.Size,
		file.MimeType,
		file.StorageKey,
		file.Hash,
		file.CreatedAt,
		file.UpdatedAt,
	)

	return err
}

// GetByID retrieves a file by its ID
func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	query := `
		SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
		       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
		FROM files
		WHERE id = $1
	`

	file := &models.File{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&file.ID,
		&file.ParentID,
		&file.OwnerID,
		&file.Name,
		&file.IsFolder,
		&file.Size,
		&file.MimeType,
		&file.StorageKey,
		&file.Hash,
		&file.IsStarred,
		&file.IsTrashed,
		&file.TrashedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.AccessedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}

	return file, err
}

// GetByName retrieves a file by name within a parent folder
func (r *FileRepository) GetByName(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, name string) (*models.File, error) {
	var query string
	var args []interface{}

	if parentID == nil {
		query = `
			SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
			       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
			FROM files
			WHERE owner_id = $1 AND parent_id IS NULL AND name = $2 AND is_trashed = false
		`
		args = []interface{}{ownerID, name}
	} else {
		query = `
			SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
			       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
			FROM files
			WHERE owner_id = $1 AND parent_id = $2 AND name = $3 AND is_trashed = false
		`
		args = []interface{}{ownerID, parentID, name}
	}

	file := &models.File{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&file.ID,
		&file.ParentID,
		&file.OwnerID,
		&file.Name,
		&file.IsFolder,
		&file.Size,
		&file.MimeType,
		&file.StorageKey,
		&file.Hash,
		&file.IsStarred,
		&file.IsTrashed,
		&file.TrashedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.AccessedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}

	return file, err
}

// ListByParent retrieves all files in a folder
func (r *FileRepository) ListByParent(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, includeTrash bool) ([]*models.File, error) {
	var query string
	var args []interface{}

	if parentID == nil {
		query = `
			SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
			       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
			FROM files
			WHERE owner_id = $1 AND parent_id IS NULL
		`
		args = []interface{}{ownerID}
	} else {
		query = `
			SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
			       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
			FROM files
			WHERE owner_id = $1 AND parent_id = $2
		`
		args = []interface{}{ownerID, parentID}
	}

	if !includeTrash {
		query += " AND is_trashed = false"
	}

	query += " ORDER BY is_folder DESC, name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*models.File, 0)
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(
			&file.ID,
			&file.ParentID,
			&file.OwnerID,
			&file.Name,
			&file.IsFolder,
			&file.Size,
			&file.MimeType,
			&file.StorageKey,
			&file.Hash,
			&file.IsStarred,
			&file.IsTrashed,
			&file.TrashedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.AccessedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// Update modifies an existing file
func (r *FileRepository) Update(ctx context.Context, file *models.File) error {
	query := `
		UPDATE files
		SET parent_id = $2, name = $3, is_starred = $4, storage_key = $5, size = $6, hash = $7, updated_at = $8
		WHERE id = $1
	`

	file.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		file.ID,
		file.ParentID,
		file.Name,
		file.IsStarred,
		file.StorageKey,
		file.Size,
		file.Hash,
		file.UpdatedAt,
	)

	return err
}

// MoveToTrash marks a file as trashed
func (r *FileRepository) MoveToTrash(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE files
		SET is_trashed = true, trashed_at = $2, updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.Exec(ctx, query, id, now)
	return err
}

// RestoreFromTrash removes the trashed flag
func (r *FileRepository) RestoreFromTrash(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE files
		SET is_trashed = false, trashed_at = NULL, updated_at = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// PermanentDelete removes a file from the database
func (r *FileRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ListTrashed retrieves all trashed files for a user
func (r *FileRepository) ListTrashed(ctx context.Context, ownerID uuid.UUID) ([]*models.File, error) {
	query := `
		SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
		       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
		FROM files
		WHERE owner_id = $1 AND is_trashed = true
		ORDER BY trashed_at DESC
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*models.File, 0)
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(
			&file.ID,
			&file.ParentID,
			&file.OwnerID,
			&file.Name,
			&file.IsFolder,
			&file.Size,
			&file.MimeType,
			&file.StorageKey,
			&file.Hash,
			&file.IsStarred,
			&file.IsTrashed,
			&file.TrashedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.AccessedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// ListStarred retrieves all starred files for a user
func (r *FileRepository) ListStarred(ctx context.Context, ownerID uuid.UUID) ([]*models.File, error) {
	query := `
		SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
		       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
		FROM files
		WHERE owner_id = $1 AND is_starred = true AND is_trashed = false
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*models.File, 0)
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(
			&file.ID,
			&file.ParentID,
			&file.OwnerID,
			&file.Name,
			&file.IsFolder,
			&file.Size,
			&file.MimeType,
			&file.StorageKey,
			&file.Hash,
			&file.IsStarred,
			&file.IsTrashed,
			&file.TrashedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.AccessedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// Search performs a text search on file names
func (r *FileRepository) Search(ctx context.Context, ownerID uuid.UUID, query string, limit int) ([]*models.File, error) {
	sqlQuery := `
		SELECT id, parent_id, owner_id, name, is_folder, size, mime_type, storage_key, hash,
		       is_starred, is_trashed, trashed_at, created_at, updated_at, accessed_at
		FROM files
		WHERE owner_id = $1 AND is_trashed = false
		  AND name ILIKE '%' || $2 || '%'
		ORDER BY similarity(name, $2) DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, sqlQuery, ownerID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*models.File, 0)
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(
			&file.ID,
			&file.ParentID,
			&file.OwnerID,
			&file.Name,
			&file.IsFolder,
			&file.Size,
			&file.MimeType,
			&file.StorageKey,
			&file.Hash,
			&file.IsStarred,
			&file.IsTrashed,
			&file.TrashedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.AccessedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// GetStorageUsed calculates total storage used by a user
func (r *FileRepository) GetStorageUsed(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	query := `
		SELECT COALESCE(SUM(size), 0)
		FROM files
		WHERE owner_id = $1 AND is_folder = false
	`

	var total int64
	err := r.db.QueryRow(ctx, query, ownerID).Scan(&total)
	return total, err
}

// GetStorageByType returns storage breakdown by mime type category
func (r *FileRepository) GetStorageByType(ctx context.Context, ownerID uuid.UUID) (map[string]int64, error) {
	query := `
		SELECT 
			CASE 
				WHEN mime_type LIKE 'image/%' THEN 'images'
				WHEN mime_type LIKE 'video/%' THEN 'videos'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'application/pdf' THEN 'documents'
				WHEN mime_type LIKE 'application/msword%' OR mime_type LIKE 'application/vnd.openxmlformats%' THEN 'documents'
				ELSE 'other'
			END AS category,
			COALESCE(SUM(size), 0) AS total
		FROM files
		WHERE owner_id = $1 AND is_folder = false AND is_trashed = false
		GROUP BY category
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var category string
		var total int64
		if err := rows.Scan(&category, &total); err != nil {
			return nil, err
		}
		result[category] = total
	}

	return result, rows.Err()
}

// CreateVersion creates a new version of a file
func (r *FileRepository) CreateVersion(ctx context.Context, version *models.FileVersion) error {
	query := `
		INSERT INTO file_versions (id, file_id, version, size, storage_key, hash, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	version.ID = uuid.New()
	version.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		version.ID,
		version.FileID,
		version.Version,
		version.Size,
		version.StorageKey,
		version.Hash,
		version.CreatedAt,
		version.CreatedBy,
	)

	return err
}

// GetVersions retrieves all versions of a file
func (r *FileRepository) GetVersions(ctx context.Context, fileID uuid.UUID) ([]*models.FileVersion, error) {
	query := `
		SELECT id, file_id, version, size, storage_key, hash, created_at, created_by
		FROM file_versions
		WHERE file_id = $1
		ORDER BY version DESC
	`

	rows, err := r.db.Query(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]*models.FileVersion, 0)
	for rows.Next() {
		v := &models.FileVersion{}
		err := rows.Scan(
			&v.ID,
			&v.FileID,
			&v.Version,
			&v.Size,
			&v.StorageKey,
			&v.Hash,
			&v.CreatedAt,
			&v.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// GetVersion retrieves a specific version of a file
func (r *FileRepository) GetVersion(ctx context.Context, fileID uuid.UUID, version int) (*models.FileVersion, error) {
	query := `
		SELECT id, file_id, version, size, storage_key, hash, created_at, created_by
		FROM file_versions
		WHERE file_id = $1 AND version = $2
	`

	v := &models.FileVersion{}
	err := r.db.QueryRow(ctx, query, fileID, version).Scan(
		&v.ID,
		&v.FileID,
		&v.Version,
		&v.Size,
		&v.StorageKey,
		&v.Hash,
		&v.CreatedAt,
		&v.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetNextVersion returns the next version number for a file
func (r *FileRepository) GetNextVersion(ctx context.Context, fileID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) + 1 FROM file_versions WHERE file_id = $1`

	var nextVersion int
	err := r.db.QueryRow(ctx, query, fileID).Scan(&nextVersion)
	return nextVersion, err
}

// CreateShare creates a new share record
func (r *FileRepository) CreateShare(ctx context.Context, share *models.Share) error {
	query := `
		INSERT INTO shares (id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, view_count, last_accessed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	share.ID = uuid.New()
	share.CreatedAt = time.Now()

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
		share.ViewCount,
		share.LastAccessedAt,
		share.CreatedAt,
	)

	return err
}

// GetShareByToken retrieves a share by its public token
func (r *FileRepository) GetShareByToken(ctx context.Context, token string) (*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, view_count, last_accessed_at, created_at
		FROM shares
		WHERE public_token = $1
	`

	share := &models.Share{}
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
		&share.ViewCount,
		&share.LastAccessedAt,
		&share.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("share not found")
	}

	return share, err
}

// IncrementShareDownloadCount increments the download count for a share
func (r *FileRepository) IncrementShareDownloadCount(ctx context.Context, shareID uuid.UUID) error {
	query := `UPDATE shares SET download_count = download_count + 1, last_accessed_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, shareID)
	return err
}

// IncrementDownloadIfAllowed atomically checks max_downloads and increments download_count
// Returns true if download was allowed and count was incremented
func (r *FileRepository) IncrementDownloadIfAllowed(ctx context.Context, shareID uuid.UUID) (bool, error) {
	query := `
		UPDATE shares 
		SET download_count = download_count + 1, last_accessed_at = NOW()
		WHERE id = $1 
		AND (max_downloads IS NULL OR download_count < max_downloads)
	`
	result, err := r.db.Exec(ctx, query, shareID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// IncrementShareViewCount increments the view count for a share
func (r *FileRepository) IncrementShareViewCount(ctx context.Context, shareID uuid.UUID) error {
	query := `UPDATE shares SET view_count = view_count + 1, last_accessed_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, shareID)
	return err
}

// GetUserShare retrieves a share for a specific user on a specific file
func (r *FileRepository) GetUserShare(ctx context.Context, fileID, userID uuid.UUID) (*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, view_count, last_accessed_at, created_at
		FROM shares
		WHERE file_id = $1 AND shared_with = $2
	`

	share := &models.Share{}
	err := r.db.QueryRow(ctx, query, fileID, userID).Scan(
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
		&share.ViewCount,
		&share.LastAccessedAt,
		&share.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	return share, err
}

// UpdateShare updates a share record
func (r *FileRepository) UpdateShare(ctx context.Context, share *models.Share) error {
	query := `UPDATE shares SET permission = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, share.Permission, share.ID)
	return err
}

// GetShareByID retrieves a share by its ID
func (r *FileRepository) GetShareByID(ctx context.Context, shareID uuid.UUID) (*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, view_count, last_accessed_at, created_at
		FROM shares
		WHERE id = $1
	`

	share := &models.Share{}
	err := r.db.QueryRow(ctx, query, shareID).Scan(
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
		&share.ViewCount,
		&share.LastAccessedAt,
		&share.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("share not found")
	}

	return share, err
}

// DeleteShare deletes a share record
func (r *FileRepository) DeleteShare(ctx context.Context, shareID uuid.UUID) error {
	query := `DELETE FROM shares WHERE id = $1`
	_, err := r.db.Exec(ctx, query, shareID)
	return err
}

// GetSharesByFile returns all shares for a file
func (r *FileRepository) GetSharesByFile(ctx context.Context, fileID uuid.UUID) ([]*models.Share, error) {
	query := `
		SELECT id, file_id, owner_id, shared_with, public_token, permission, password_hash, expires_at, max_downloads, download_count, view_count, last_accessed_at, created_at
		FROM shares
		WHERE file_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := make([]*models.Share, 0)
	for rows.Next() {
		share := &models.Share{}
		err := rows.Scan(
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
			&share.ViewCount,
			&share.LastAccessedAt,
			&share.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}

	return shares, rows.Err()
}

// GetSharedWithUser returns all files shared with a user
func (r *FileRepository) GetSharedWithUser(ctx context.Context, userID uuid.UUID) ([]*models.SharedFile, error) {
	query := `
		SELECT f.id, f.name, f.is_folder, f.size, f.mime_type, s.permission, 
		       u.id as owner_id, u.name as owner_name, u.email as owner_email, s.created_at as shared_at
		FROM shares s
		JOIN files f ON f.id = s.file_id
		JOIN users u ON u.id = s.owner_id
		WHERE s.shared_with = $1 AND f.is_trashed = false
		ORDER BY s.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*models.SharedFile, 0)
	for rows.Next() {
		sf := &models.SharedFile{}
		err := rows.Scan(
			&sf.ID,
			&sf.Name,
			&sf.IsFolder,
			&sf.Size,
			&sf.MimeType,
			&sf.Permission,
			&sf.OwnerID,
			&sf.OwnerName,
			&sf.OwnerEmail,
			&sf.SharedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, sf)
	}

	return files, rows.Err()
}
