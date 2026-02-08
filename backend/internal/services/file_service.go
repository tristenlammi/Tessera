package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/storage"
)

var (
	// ErrQuotaExceeded is returned when upload would exceed user's storage quota
	ErrQuotaExceeded = errors.New("storage quota exceeded")
)

// FileService handles file operations
type FileService struct {
	fileRepo *repository.FileRepository
	userRepo *repository.UserRepository
	storage  *storage.MinIOStorage
	log      zerolog.Logger
}

// NewFileService creates a new file service
func NewFileService(fileRepo *repository.FileRepository, userRepo *repository.UserRepository, storage *storage.MinIOStorage, log zerolog.Logger) *FileService {
	return &FileService{
		fileRepo: fileRepo,
		userRepo: userRepo,
		storage:  storage,
		log:      log,
	}
}

// GetUserByEmail looks up a user by email
func (s *FileService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// ListInput contains parameters for listing files
type ListInput struct {
	OwnerID  uuid.UUID
	ParentID *uuid.UUID
}

// List retrieves files in a folder
func (s *FileService) List(ctx context.Context, input ListInput) ([]*models.File, error) {
	return s.fileRepo.ListByParent(ctx, input.OwnerID, input.ParentID, false)
}

// Get retrieves a single file by ID
func (s *FileService) Get(ctx context.Context, fileID, ownerID uuid.UUID) (*models.File, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	// Verify ownership (TODO: check shares)
	if file.OwnerID != ownerID {
		return nil, repository.ErrFileNotFound
	}

	return file, nil
}

// CreateFolderInput contains folder creation data
type CreateFolderInput struct {
	OwnerID  uuid.UUID
	ParentID *uuid.UUID
	Name     string
}

// CreateFolder creates a new folder
func (s *FileService) CreateFolder(ctx context.Context, input CreateFolderInput) (*models.File, error) {
	folder := &models.File{
		ParentID: input.ParentID,
		OwnerID:  input.OwnerID,
		Name:     input.Name,
		IsFolder: true,
	}

	if err := s.fileRepo.Create(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

// UploadInput contains upload metadata
type UploadInput struct {
	OwnerID  uuid.UUID
	ParentID *uuid.UUID
	Name     string
	Size     int64
	Reader   io.Reader
}

// UploadFile stores a new file
func (s *FileService) UploadFile(ctx context.Context, input UploadInput) (*models.File, error) {
	// Check storage quota before uploading
	user, err := s.userRepo.GetByID(ctx, input.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Enforce quota if limit is set (> 0)
	if user.StorageLimit > 0 && user.StorageUsed+input.Size > user.StorageLimit {
		return nil, ErrQuotaExceeded
	}

	// Determine MIME type from extension
	ext := filepath.Ext(input.Name)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Generate storage key
	storageKey := fmt.Sprintf("%s/%s/%s", input.OwnerID.String(), time.Now().Format("2006/01/02"), uuid.New().String())

	// Calculate hash while uploading (using TeeReader would be ideal here)
	// For simplicity, we'll upload first, then the hash would be calculated during chunked upload
	hash := ""

	// Upload to storage
	if err := s.storage.Upload(ctx, storageKey, input.Reader, input.Size, mimeType); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Create file record
	file := &models.File{
		ParentID:   input.ParentID,
		OwnerID:    input.OwnerID,
		Name:       input.Name,
		IsFolder:   false,
		Size:       input.Size,
		MimeType:   mimeType,
		StorageKey: storageKey,
		Hash:       hash,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// Cleanup uploaded file on error
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return file, nil
}

// UpdateInput contains file update data
type UpdateInput struct {
	FileID    uuid.UUID
	OwnerID   uuid.UUID
	Name      *string
	ParentID  *uuid.UUID
	IsStarred *bool
}

// UpdateContentInput contains file content update data
type UpdateContentInput struct {
	FileID  uuid.UUID
	OwnerID uuid.UUID
	Reader  io.Reader
	Size    int64
}

// Update modifies a file's metadata
func (s *FileService) Update(ctx context.Context, input UpdateInput) (*models.File, error) {
	file, err := s.Get(ctx, input.FileID, input.OwnerID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		file.Name = *input.Name
	}
	if input.ParentID != nil {
		file.ParentID = input.ParentID
	}
	if input.IsStarred != nil {
		file.IsStarred = *input.IsStarred
	}

	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, err
	}

	return file, nil
}

// Delete moves a file to trash
func (s *FileService) Delete(ctx context.Context, fileID, ownerID uuid.UUID) error {
	file, err := s.Get(ctx, fileID, ownerID)
	if err != nil {
		return err
	}

	return s.fileRepo.MoveToTrash(ctx, file.ID)
}

// Restore recovers a file from trash
func (s *FileService) Restore(ctx context.Context, fileID, ownerID uuid.UUID) (*models.File, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if file.OwnerID != ownerID {
		return nil, repository.ErrFileNotFound
	}

	if err := s.fileRepo.RestoreFromTrash(ctx, file.ID); err != nil {
		return nil, err
	}

	file.IsTrashed = false
	file.TrashedAt = nil

	return file, nil
}

// PermanentDelete removes a file permanently
func (s *FileService) PermanentDelete(ctx context.Context, fileID, ownerID uuid.UUID) error {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	if file.OwnerID != ownerID {
		return repository.ErrFileNotFound
	}

	// Delete from storage
	if !file.IsFolder && file.StorageKey != "" {
		if err := s.storage.Delete(ctx, file.StorageKey); err != nil {
			s.log.Error().Err(err).Str("storage_key", file.StorageKey).Msg("Failed to delete file from storage")
		}
	}

	return s.fileRepo.PermanentDelete(ctx, file.ID)
}

// CopyFile duplicates a file
func (s *FileService) CopyFile(ctx context.Context, fileID, ownerID uuid.UUID, destParentID *uuid.UUID, newName string) (*models.File, error) {
	source, err := s.Get(ctx, fileID, ownerID)
	if err != nil {
		return nil, err
	}

	if source.IsFolder {
		// TODO: Implement recursive folder copy
		return nil, fmt.Errorf("folder copy not yet implemented")
	}

	// Download source file
	reader, err := s.storage.Download(ctx, source.StorageKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	name := source.Name
	if newName != "" {
		name = newName
	}

	// Upload as new file
	return s.UploadFile(ctx, UploadInput{
		OwnerID:  ownerID,
		ParentID: destParentID,
		Name:     name,
		Size:     source.Size,
		Reader:   reader,
	})
}

// Download returns a reader for a file's content
func (s *FileService) Download(ctx context.Context, fileID, ownerID uuid.UUID) (io.ReadCloser, *models.File, error) {
	file, err := s.Get(ctx, fileID, ownerID)
	if err != nil {
		return nil, nil, err
	}

	if file.IsFolder {
		return nil, nil, fmt.Errorf("cannot download a folder")
	}

	reader, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return reader, file, nil
}

// GetPresignedURL returns a temporary download URL
func (s *FileService) GetPresignedURL(ctx context.Context, fileID, ownerID uuid.UUID, expiry time.Duration) (string, error) {
	file, err := s.Get(ctx, fileID, ownerID)
	if err != nil {
		return "", err
	}

	return s.storage.GetPresignedURL(ctx, file.StorageKey, expiry)
}

// ListTrash retrieves all trashed files
func (s *FileService) ListTrash(ctx context.Context, ownerID uuid.UUID) ([]*models.File, error) {
	return s.fileRepo.ListTrashed(ctx, ownerID)
}

// ListStarred retrieves all starred files
func (s *FileService) ListStarred(ctx context.Context, ownerID uuid.UUID) ([]*models.File, error) {
	return s.fileRepo.ListStarred(ctx, ownerID)
}

// EmptyTrash permanently deletes all trashed files
func (s *FileService) EmptyTrash(ctx context.Context, ownerID uuid.UUID) error {
	files, err := s.fileRepo.ListTrashed(ctx, ownerID)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := s.PermanentDelete(ctx, file.ID, ownerID); err != nil {
			s.log.Error().Err(err).Str("file_id", file.ID.String()).Msg("Failed to delete file during empty trash")
		}
	}

	return nil
}

// Search finds files matching a query
func (s *FileService) Search(ctx context.Context, ownerID uuid.UUID, query string, limit int) ([]*models.File, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.fileRepo.Search(ctx, ownerID, query, limit)
}

// StorageStats represents storage usage statistics
type StorageStats struct {
	Used    int64            `json:"used"`
	Limit   int64            `json:"limit"`
	ByType  map[string]int64 `json:"by_type"`
	UsedPct float64          `json:"used_pct"`
}

// GetStorageStats returns storage usage statistics
func (s *FileService) GetStorageStats(ctx context.Context, ownerID uuid.UUID, storageLimit int64) (*StorageStats, error) {
	used, err := s.fileRepo.GetStorageUsed(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	byType, err := s.fileRepo.GetStorageByType(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	usedPct := 0.0
	if storageLimit > 0 {
		usedPct = float64(used) / float64(storageLimit) * 100
	}

	return &StorageStats{
		Used:    used,
		Limit:   storageLimit,
		ByType:  byType,
		UsedPct: usedPct,
	}, nil
}

// GetVersions retrieves version history for a file
func (s *FileService) GetVersions(ctx context.Context, fileID, ownerID uuid.UUID) ([]*models.FileVersion, error) {
	// Verify ownership
	if _, err := s.Get(ctx, fileID, ownerID); err != nil {
		return nil, err
	}

	return s.fileRepo.GetVersions(ctx, fileID)
}

// RestoreVersion restores a file to a specific version
func (s *FileService) RestoreVersion(ctx context.Context, fileID, ownerID uuid.UUID, version int) (*models.File, error) {
	// Verify ownership
	file, err := s.Get(ctx, fileID, ownerID)
	if err != nil {
		return nil, err
	}

	// Get the version to restore
	v, err := s.fileRepo.GetVersion(ctx, fileID, version)
	if err != nil {
		return nil, err
	}

	// Save current file as a new version before restoring
	nextVersion, err := s.fileRepo.GetNextVersion(ctx, fileID)
	if err != nil {
		return nil, err
	}

	currentVersion := &models.FileVersion{
		FileID:     fileID,
		Version:    nextVersion,
		Size:       file.Size,
		StorageKey: file.StorageKey,
		Hash:       file.Hash,
		CreatedBy:  ownerID,
	}
	if err := s.fileRepo.CreateVersion(ctx, currentVersion); err != nil {
		return nil, err
	}

	// Update file with the restored version's data
	file.Size = v.Size
	file.StorageKey = v.StorageKey
	file.Hash = v.Hash
	file.UpdatedAt = time.Now()

	if err := s.fileRepo.Update(ctx, file); err != nil {
		return nil, err
	}

	return file, nil
}

// CreateShareInput contains share creation data
type CreateShareInput struct {
	FileID        uuid.UUID
	OwnerID       uuid.UUID
	ExpiresInDays *int
	Password      *string
	AllowDownload *bool
	MaxDownloads  *int
}

// ShareResponse represents the share info returned to client
type ShareResponse struct {
	Token         string     `json:"token"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	AllowDownload bool       `json:"allow_download"`
	MaxDownloads  *int       `json:"max_downloads,omitempty"`
}

// CreateShare creates a share link for a file
func (s *FileService) CreateShare(ctx context.Context, input CreateShareInput) (*ShareResponse, error) {
	// Verify file exists and user owns it
	file, err := s.Get(ctx, input.FileID, input.OwnerID)
	if err != nil {
		return nil, err
	}

	// Generate unique token
	token := uuid.New().String()[:12]

	// Calculate expiration
	var expiresAt *time.Time
	if input.ExpiresInDays != nil && *input.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(*input.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}

	// Default allow download to true
	allowDownload := true
	if input.AllowDownload != nil {
		allowDownload = *input.AllowDownload
	}

	// Hash password if provided using bcrypt
	var passwordHash *string
	if input.Password != nil && *input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashStr := string(hash)
		passwordHash = &hashStr
	}

	share := &models.Share{
		FileID:       file.ID,
		OwnerID:      input.OwnerID,
		PublicToken:  &token,
		Permission:   "download",
		PasswordHash: passwordHash,
		ExpiresAt:    expiresAt,
		MaxDownloads: input.MaxDownloads,
	}

	// Set permission based on allow download
	if !allowDownload {
		share.Permission = "view"
	}

	if err := s.fileRepo.CreateShare(ctx, share); err != nil {
		return nil, err
	}

	return &ShareResponse{
		Token:         token,
		ExpiresAt:     expiresAt,
		AllowDownload: allowDownload,
		MaxDownloads:  input.MaxDownloads,
	}, nil
}

// ShareInfo contains share metadata
type ShareInfo struct {
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	IsFolder      bool   `json:"is_folder"`
	AllowDownload bool   `json:"allow_download"`
	HasPassword   bool   `json:"has_password"`
	MaxDownloads  *int   `json:"max_downloads,omitempty"`
	DownloadsLeft *int   `json:"downloads_left,omitempty"`
}

// GetShare retrieves share info by token and increments view count
func (s *FileService) GetShare(ctx context.Context, token string) (*ShareInfo, error) {
	share, err := s.fileRepo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("share expired")
	}

	// Check max downloads if set
	if share.MaxDownloads != nil && share.DownloadCount >= *share.MaxDownloads {
		return nil, fmt.Errorf("download limit reached")
	}

	file, err := s.fileRepo.GetByID(ctx, share.FileID)
	if err != nil {
		return nil, err
	}

	// Increment view count
	_ = s.fileRepo.IncrementShareViewCount(ctx, share.ID)

	allowDownload := share.Permission == "download" || share.Permission == "edit"

	info := &ShareInfo{
		FileName:      file.Name,
		FileSize:      file.Size,
		IsFolder:      file.IsFolder,
		AllowDownload: allowDownload,
		HasPassword:   share.PasswordHash != nil,
		MaxDownloads:  share.MaxDownloads,
	}

	// Calculate downloads left
	if share.MaxDownloads != nil {
		left := *share.MaxDownloads - share.DownloadCount
		info.DownloadsLeft = &left
	}

	return info, nil
}

// ShareAnalytics contains analytics data for a share
type ShareAnalytics struct {
	Token          string     `json:"token"`
	ViewCount      int        `json:"view_count"`
	DownloadCount  int        `json:"download_count"`
	MaxDownloads   *int       `json:"max_downloads,omitempty"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	HasPassword    bool       `json:"has_password"`
	AllowDownload  bool       `json:"allow_download"`
}

// GetShareAnalytics retrieves analytics for a share owned by the user
func (s *FileService) GetShareAnalytics(ctx context.Context, fileID, ownerID uuid.UUID) (*ShareAnalytics, error) {
	// Get shares for this file
	shares, err := s.fileRepo.GetSharesByFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	// Find the public link share owned by this user
	for _, share := range shares {
		if share.OwnerID == ownerID && share.PublicToken != nil {
			return &ShareAnalytics{
				Token:          *share.PublicToken,
				ViewCount:      share.ViewCount,
				DownloadCount:  share.DownloadCount,
				MaxDownloads:   share.MaxDownloads,
				LastAccessedAt: share.LastAccessedAt,
				CreatedAt:      share.CreatedAt,
				ExpiresAt:      share.ExpiresAt,
				HasPassword:    share.PasswordHash != nil,
				AllowDownload:  share.Permission == "download" || share.Permission == "edit",
			}, nil
		}
	}

	return nil, fmt.Errorf("share not found")
}

// FileInfo contains basic file metadata for downloads
type FileInfo struct {
	Name     string
	MimeType string
}

// DownloadShare downloads a shared file
func (s *FileService) DownloadShare(ctx context.Context, token, password string) (io.ReadCloser, *FileInfo, error) {
	share, err := s.fileRepo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	// Check if expired
	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		return nil, nil, fmt.Errorf("share expired")
	}

	// Check download permission
	allowDownload := share.Permission == "download" || share.Permission == "edit"
	if !allowDownload {
		return nil, nil, fmt.Errorf("download not allowed")
	}

	// Check password using bcrypt
	if share.PasswordHash != nil {
		if password == "" {
			return nil, nil, fmt.Errorf("password required")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(password)); err != nil {
			return nil, nil, fmt.Errorf("invalid password")
		}
	}

	// Atomically check max downloads and increment count
	// This prevents race conditions where multiple downloads could exceed the limit
	allowed, err := s.fileRepo.IncrementDownloadIfAllowed(ctx, share.ID)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, fmt.Errorf("max downloads reached")
	}

	file, err := s.fileRepo.GetByID(ctx, share.FileID)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return reader, &FileInfo{
		Name:     file.Name,
		MimeType: file.MimeType,
	}, nil
}

// calculateHash computes SHA-256 hash of data
func calculateHash(reader io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SharePermission constants
const (
	PermissionView  = "view"
	PermissionEdit  = "edit"
	PermissionAdmin = "admin"
)

// ShareWithUserInput contains user share creation data
type ShareWithUserInput struct {
	FileID     uuid.UUID
	OwnerID    uuid.UUID
	SharedWith uuid.UUID
	Permission string
}

// UserShareResponse represents user share info
type UserShareResponse struct {
	ID          uuid.UUID `json:"id"`
	FileID      uuid.UUID `json:"file_id"`
	FileName    string    `json:"file_name"`
	SharedWith  uuid.UUID `json:"shared_with"`
	SharedName  string    `json:"shared_name"`
	SharedEmail string    `json:"shared_email"`
	Permission  string    `json:"permission"`
	CreatedAt   time.Time `json:"created_at"`
}

// ShareWithUser shares a file with another user
func (s *FileService) ShareWithUser(ctx context.Context, input ShareWithUserInput) (*UserShareResponse, error) {
	// Verify file exists and user owns it
	file, err := s.Get(ctx, input.FileID, input.OwnerID)
	if err != nil {
		return nil, err
	}

	// Validate permission
	if input.Permission != PermissionView && input.Permission != PermissionEdit && input.Permission != PermissionAdmin {
		input.Permission = PermissionView
	}

	// Check if already shared with this user
	existing, _ := s.fileRepo.GetUserShare(ctx, input.FileID, input.SharedWith)
	if existing != nil {
		// Update existing share
		existing.Permission = input.Permission
		if err := s.fileRepo.UpdateShare(ctx, existing); err != nil {
			return nil, err
		}

		return &UserShareResponse{
			ID:         existing.ID,
			FileID:     file.ID,
			FileName:   file.Name,
			SharedWith: input.SharedWith,
			Permission: input.Permission,
			CreatedAt:  existing.CreatedAt,
		}, nil
	}

	// Create new share
	share := &models.Share{
		FileID:     file.ID,
		OwnerID:    input.OwnerID,
		SharedWith: &input.SharedWith,
		Permission: input.Permission,
	}

	if err := s.fileRepo.CreateShare(ctx, share); err != nil {
		return nil, err
	}

	return &UserShareResponse{
		ID:         share.ID,
		FileID:     file.ID,
		FileName:   file.Name,
		SharedWith: input.SharedWith,
		Permission: input.Permission,
		CreatedAt:  share.CreatedAt,
	}, nil
}

// GetFileShares returns all shares for a file
func (s *FileService) GetFileShares(ctx context.Context, fileID, ownerID uuid.UUID) ([]*models.Share, error) {
	// Verify ownership
	if _, err := s.Get(ctx, fileID, ownerID); err != nil {
		return nil, err
	}

	return s.fileRepo.GetSharesByFile(ctx, fileID)
}

// GetSharedWithMe returns files shared with a user
func (s *FileService) GetSharedWithMe(ctx context.Context, userID uuid.UUID) ([]*models.SharedFile, error) {
	return s.fileRepo.GetSharedWithUser(ctx, userID)
}

// RevokeUserShare removes a user's share access
func (s *FileService) RevokeUserShare(ctx context.Context, shareID, ownerID uuid.UUID) error {
	share, err := s.fileRepo.GetShareByID(ctx, shareID)
	if err != nil {
		return err
	}

	// Verify owner
	if share.OwnerID != ownerID {
		return fmt.Errorf("not authorized")
	}

	return s.fileRepo.DeleteShare(ctx, shareID)
}

// CanUserAccessFile checks if a user can access a file (owner or shared)
func (s *FileService) CanUserAccessFile(ctx context.Context, fileID, userID uuid.UUID) (bool, string, error) {
	// Check if owner
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return false, "", err
	}
	if file.OwnerID == userID {
		return true, PermissionAdmin, nil
	}

	// Check if shared
	share, err := s.fileRepo.GetUserShare(ctx, fileID, userID)
	if err != nil || share == nil {
		return false, "", nil
	}

	return true, share.Permission, nil
}

// WebDAV helper methods that accept string IDs

// UpdateFileContent updates the content of an existing file
func (s *FileService) UpdateFileContent(ctx context.Context, fileID, userID string, reader io.Reader, size int64) (*models.File, error) {
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file ID")
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Get existing file
	file, err := s.Get(ctx, fileUUID, userUUID)
	if err != nil {
		return nil, err
	}

	if file.IsFolder {
		return nil, fmt.Errorf("cannot update content of a folder")
	}

	// Create version of old file
	if file.StorageKey != "" {
		version := &models.FileVersion{
			FileID:     file.ID,
			Version:    1, // Will be auto-incremented
			Size:       file.Size,
			StorageKey: file.StorageKey,
			Hash:       file.Hash,
			CreatedBy:  userUUID,
		}
		// Get next version number
		versions, _ := s.fileRepo.GetVersions(ctx, file.ID)
		if len(versions) > 0 {
			version.Version = versions[0].Version + 1
		}
		_ = s.fileRepo.CreateVersion(ctx, version)
	}

	// Upload new content
	newStorageKey := fmt.Sprintf("%s/%s/%s", userID, time.Now().Format("2006/01/02"), uuid.New().String())
	if err := s.storage.Upload(ctx, newStorageKey, reader, size, file.MimeType); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Update file record
	file.StorageKey = newStorageKey
	file.Size = size
	file.UpdatedAt = time.Now()

	if err := s.fileRepo.Update(ctx, file); err != nil {
		_ = s.storage.Delete(ctx, newStorageKey)
		return nil, err
	}

	return file, nil
}

// UpdateFileContentWithInput updates file content using typed input
func (s *FileService) UpdateFileContentWithInput(ctx context.Context, input UpdateContentInput) (*models.File, error) {
	return s.UpdateFileContent(ctx, input.FileID.String(), input.OwnerID.String(), input.Reader, input.Size)
}

// UploadString is a WebDAV helper that accepts string IDs
func (s *FileService) Upload(ctx context.Context, userID, parentID, name string, reader io.Reader, size int64, mimeType string) (*models.File, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	var parentUUID *uuid.UUID
	if parentID != "" {
		parsed, err := uuid.Parse(parentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID")
		}
		parentUUID = &parsed
	}

	return s.UploadFile(ctx, UploadInput{
		OwnerID:  userUUID,
		ParentID: parentUUID,
		Name:     name,
		Size:     size,
		Reader:   reader,
	})
}

// CopyString is a WebDAV helper that accepts string IDs
func (s *FileService) Copy(ctx context.Context, fileID, userID, destParentID, newName string) (*models.File, error) {
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file ID")
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	var destParentUUID *uuid.UUID
	if destParentID != "" {
		parsed, err := uuid.Parse(destParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid destination parent ID")
		}
		destParentUUID = &parsed
	}

	return s.CopyFile(ctx, fileUUID, userUUID, destParentUUID, newName)
}
