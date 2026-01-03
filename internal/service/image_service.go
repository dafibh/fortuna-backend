package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"path/filepath"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/repository/storage"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

const (
	MaxImageSize     = 5 * 1024 * 1024 // 5MB
	MinImageWidth    = 50
	MinImageHeight   = 50
	ThumbnailWidth   = 200
	DisplayWidth     = 800
	JPEGQuality      = 85
)

var (
	ErrImageTooLarge    = errors.New("file too large. Maximum size is 5MB")
	ErrInvalidFormat    = errors.New("invalid format. Supported: JPEG, PNG, WebP")
	ErrImageTooSmall    = errors.New("image too small. Minimum 50x50 pixels")
	ErrInvalidImageData = errors.New("invalid image data")
	ErrImageStorageNotConfigured = errors.New("image storage not configured")
)

// AllowedImageFormats contains the supported image MIME types
var AllowedImageFormats = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// AllowedExtensions maps extensions to content types
var AllowedExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

// ImageMetadata contains URLs for different image sizes
type ImageMetadata struct {
	ID           string `json:"id"`
	ThumbnailURL string `json:"thumbnailUrl"`
	DisplayURL   string `json:"displayUrl"`
	OriginalURL  string `json:"originalUrl"`
}

// ImageService handles image processing and storage
type ImageService struct {
	storage storage.ImageRepository
}

// NewImageService creates a new ImageService
func NewImageService(storage storage.ImageRepository) *ImageService {
	return &ImageService{storage: storage}
}

// IsEnabled indicates whether uploads/deletes are supported (storage configured).
func (s *ImageService) IsEnabled() bool {
	return s != nil && s.storage != nil
}

// ValidateImage validates image format and size
func (s *ImageService) ValidateImage(data []byte, filename string) error {
	_, err := s.validateAndDecode(data, filename)
	return err
}

// validateAndDecode validates the image and returns the decoded image
func (s *ImageService) validateAndDecode(data []byte, filename string) (image.Image, error) {
	// Check file size
	if len(data) > MaxImageSize {
		return nil, ErrImageTooLarge
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := AllowedExtensions[ext]; !ok {
		return nil, ErrInvalidFormat
	}

	// Decode image to verify it's valid and check dimensions
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImageData
	}

	bounds := img.Bounds()
	if bounds.Dx() < MinImageWidth || bounds.Dy() < MinImageHeight {
		return nil, ErrImageTooSmall
	}

	return img, nil
}

// ProcessAndUpload processes an image (resize) and uploads all variants
func (s *ImageService) ProcessAndUpload(ctx context.Context, workspaceID int32, entityType string, entityID int32, data []byte, filename string) (*ImageMetadata, error) {
	if !s.IsEnabled() {
		return nil, ErrImageStorageNotConfigured
	}

	// Validate and decode the image in one step
	img, err := s.validateAndDecode(data, filename)
	if err != nil {
		return nil, err
	}

	// Generate unique ID for this upload
	imageID := uuid.New().String()

	// Define sizes to generate
	variants := []struct {
		name     string
		maxWidth int
	}{
		{"thumb", ThumbnailWidth},
		{"display", DisplayWidth},
		{"original", 0}, // 0 means keep original size
	}

	urls := make(map[string]string)

	for _, variant := range variants {
		var processed image.Image
		if variant.maxWidth > 0 && img.Bounds().Dx() > variant.maxWidth {
			// Resize maintaining aspect ratio
			processed = imaging.Resize(img, variant.maxWidth, 0, imaging.Lanczos)
		} else {
			processed = img
		}

		// Encode to JPEG
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, processed, &jpeg.Options{Quality: JPEGQuality}); err != nil {
			return nil, fmt.Errorf("failed to encode image: %w", err)
		}

		// Generate object path
		objectPath := fmt.Sprintf("%d/%s/%d/%s_%s.jpg", workspaceID, entityType, entityID, imageID, variant.name)

		// Upload to storage
		url, err := s.storage.Upload(ctx, objectPath, bytes.NewReader(buf.Bytes()), "image/jpeg", int64(buf.Len()))
		if err != nil {
			// Try to clean up any already uploaded variants
			s.cleanupVariants(ctx, urls)
			return nil, fmt.Errorf("failed to upload %s variant: %w", variant.name, err)
		}

		urls[variant.name] = url
	}

	return &ImageMetadata{
		ID:           imageID,
		ThumbnailURL: urls["thumb"],
		DisplayURL:   urls["display"],
		OriginalURL:  urls["original"],
	}, nil
}

// cleanupVariants removes any variants that were successfully uploaded during a failed operation
func (s *ImageService) cleanupVariants(ctx context.Context, urls map[string]string) {
	for _, url := range urls {
		// Extract path and delete - ignore errors during cleanup
		_ = s.DeleteByURL(ctx, url)
	}
}

// DeleteByURL deletes an image by its URL
func (s *ImageService) DeleteByURL(ctx context.Context, imageURL string) error {
	if imageURL == "" {
		return nil
	}
	if !s.IsEnabled() {
		return ErrImageStorageNotConfigured
	}
	if repo, ok := s.storage.(*storage.MinIOImageRepository); ok {
		return repo.DeleteByURL(ctx, imageURL)
	}
	return nil
}

// DeleteAllVariants deletes all variants of an image (thumbnail, display, original)
func (s *ImageService) DeleteAllVariants(ctx context.Context, imageURL string) error {
	if imageURL == "" {
		return nil
	}
	if !s.IsEnabled() {
		return ErrImageStorageNotConfigured
	}

	// The imageURL is typically the display URL
	// We need to extract the base path and delete all variants
	basePath := s.extractBasePath(imageURL)
	if basePath == "" {
		return nil
	}

	variants := []string{"thumb", "display", "original"}
	for _, variant := range variants {
		variantURL := basePath + "_" + variant + ".jpg"
		if err := s.DeleteByURL(ctx, variantURL); err != nil {
			// Log but don't fail - best effort cleanup
			continue
		}
	}

	return nil
}

// extractBasePath extracts the base path from an image URL (without variant suffix)
func (s *ImageService) extractBasePath(imageURL string) string {
	// URL format: http://endpoint/bucket/workspace/entity/entityId/uuid_variant.jpg
	// We want: http://endpoint/bucket/workspace/entity/entityId/uuid
	suffixes := []string{"_thumb.jpg", "_display.jpg", "_original.jpg"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(imageURL, suffix) {
			return strings.TrimSuffix(imageURL, suffix)
		}
	}
	return ""
}

// GetContentType returns the content type for a file extension
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct, ok := AllowedExtensions[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// IsValidImageFormat checks if a content type is a valid image format
func IsValidImageFormat(contentType string) bool {
	return AllowedImageFormats[contentType]
}

// ReadAll reads all data from a reader into a byte slice
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
