package storage

import (
	"context"
	"io"
	"time"
)

// ImageRepository defines the interface for image storage operations
type ImageRepository interface {
	Upload(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, objectPath string) error
	GeneratePresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
}
