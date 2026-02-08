package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/tessera/tessera/internal/config"
)

// Storage defines the interface for file storage operations
type Storage interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectName string) error
	GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)
	Stat(ctx context.Context, objectName string) (*ObjectInfo, error)
}

// ObjectInfo contains metadata about a stored object
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// MinIOStorage implements Storage interface using MinIO
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIO creates a new MinIO storage client
func NewMinIO(cfg config.StorageConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Verify bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload stores a file in MinIO
func (s *MinIOStorage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, opts)
	return err
}

// Download retrieves a file from MinIO
func (s *MinIOStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
}

// Delete removes a file from MinIO
func (s *MinIOStorage) Delete(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}

// GetPresignedURL generates a temporary download URL
func (s *MinIOStorage) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// Stat returns object metadata
func (s *MinIOStorage) Stat(ctx context.Context, objectName string) (*ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	return &ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
		ETag:         info.ETag,
	}, nil
}

// Client returns the underlying MinIO client for health checks
func (s *MinIOStorage) Client() *minio.Client {
	return s.client
}
