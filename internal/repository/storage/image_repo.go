package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ImageRepository defines the interface for image storage operations
type ImageRepository interface {
	Upload(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, objectPath string) error
	GenerateURL(objectPath string) string
}

// MinIOImageRepository implements ImageRepository using MinIO
type MinIOImageRepository struct {
	client     *minio.Client
	bucketName string
	endpoint   string
	useSSL     bool
}

// NewMinIOImageRepository creates a new MinIO image repository
func NewMinIOImageRepository(cfg config.MinIOConfig) (*MinIOImageRepository, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	repo := &MinIOImageRepository{
		client:     client,
		bucketName: cfg.BucketName,
		endpoint:   cfg.Endpoint,
		useSSL:     cfg.UseSSL,
	}

	// Ensure bucket exists
	if err := repo.ensureBucket(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (r *MinIOImageRepository) ensureBucket(ctx context.Context) error {
	exists, err := r.client.BucketExists(ctx, r.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = r.client.MakeBucket(ctx, r.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		// Set bucket policy to allow public read access
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}]
		}`, r.bucketName)

		err = r.client.SetBucketPolicy(ctx, r.bucketName, policy)
		if err != nil {
			return fmt.Errorf("failed to set bucket policy: %w", err)
		}
	}

	return nil
}

// Upload uploads data to MinIO storage
func (r *MinIOImageRepository) Upload(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error) {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	// If size is unknown, read all data into memory
	if size < 0 {
		buf, err := io.ReadAll(data)
		if err != nil {
			return "", fmt.Errorf("failed to read data: %w", err)
		}
		size = int64(len(buf))
		data = bytes.NewReader(buf)
	}

	_, err := r.client.PutObject(ctx, r.bucketName, objectPath, data, size, opts)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return r.GenerateURL(objectPath), nil
}

// Delete removes an object from MinIO storage
func (r *MinIOImageRepository) Delete(ctx context.Context, objectPath string) error {
	err := r.client.RemoveObject(ctx, r.bucketName, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// DeleteByURL extracts the object path from a URL and deletes the object
func (r *MinIOImageRepository) DeleteByURL(ctx context.Context, imageURL string) error {
	// Extract the object path from the URL
	objectPath := r.extractObjectPath(imageURL)
	if objectPath == "" {
		return nil // Nothing to delete
	}
	return r.Delete(ctx, objectPath)
}

// extractObjectPath extracts the object path from a full URL
func (r *MinIOImageRepository) extractObjectPath(imageURL string) string {
	// URL format: http(s)://endpoint/bucket/path
	// We need to extract the path after the bucket name
	prefix := r.GenerateURL("")
	if len(imageURL) > len(prefix) {
		return imageURL[len(prefix):]
	}
	return ""
}

// GenerateURL generates a public URL for an object
func (r *MinIOImageRepository) GenerateURL(objectPath string) string {
	scheme := "http"
	if r.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, r.endpoint, r.bucketName, objectPath)
}

// GenerateObjectPath creates a unique object path for an image
func GenerateObjectPath(workspaceID int32, entityType string, entityID int32, variant string, ext string) string {
	id := uuid.New().String()
	filename := fmt.Sprintf("%s_%s%s", id, variant, ext)
	return path.Join(fmt.Sprintf("%d", workspaceID), entityType, fmt.Sprintf("%d", entityID), filename)
}

// GeneratePresignedURL generates a presigned URL for temporary access
func (r *MinIOImageRepository) GeneratePresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	presignedURL, err := r.client.PresignedGetObject(ctx, r.bucketName, objectPath, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}
