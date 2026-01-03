package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ImageHandler handles image-related HTTP requests
type ImageHandler struct {
	imageService *service.ImageService
}

// NewImageHandler creates a new ImageHandler
func NewImageHandler(imageService *service.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// UploadImageRequest represents the upload request parameters
type UploadImageRequest struct {
	EntityType string `form:"entityType" query:"entityType"`
	EntityID   int32  `form:"entityId" query:"entityId"`
}

// UploadImageResponse represents the upload response
type UploadImageResponse struct {
	ID           string `json:"id"`
	ThumbnailURL string `json:"thumbnailUrl"`
	DisplayURL   string `json:"displayUrl"`
	OriginalURL  string `json:"originalUrl"`
}

// UploadImage handles POST /api/v1/images
func (h *ImageHandler) UploadImage(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	// If storage isn't configured, don't attempt to process/upload (would panic on nil storage).
	if h.imageService == nil || !h.imageService.IsEnabled() {
		return NewServiceUnavailableError(c, "Image uploads are disabled (storage not configured)")
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return NewValidationError(c, "No file provided", []ValidationError{
			{Field: "file", Message: "File is required"},
		})
	}

	// Get entity parameters
	entityType := c.FormValue("entityType")
	if entityType == "" {
		entityType = "notes" // Default to notes
	}

	// Validate entity type
	validEntityTypes := map[string]bool{
		"notes":  true,
		"items":  true,
		"prices": true,
	}
	if !validEntityTypes[entityType] {
		return NewValidationError(c, "Invalid entity type", []ValidationError{
			{Field: "entityType", Message: "Must be one of: notes, items, prices"},
		})
	}

	var entityID int32 = 0 // Will be associated later when saving the note/item/price

	// Open the file
	src, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open uploaded file")
		return NewInternalError(c, "Failed to process file")
	}
	defer src.Close()

	// Read all data for processing
	data, err := io.ReadAll(src)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read uploaded file")
		return NewInternalError(c, "Failed to read file")
	}

	// Process and upload image
	metadata, err := h.imageService.ProcessAndUpload(c.Request().Context(), workspaceID, entityType, entityID, data, file.Filename)
	if err != nil {
		switch err {
		case service.ErrImageTooLarge:
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "file", Message: "File too large. Maximum size is 5MB"},
			})
		case service.ErrInvalidFormat:
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "file", Message: "Invalid format. Supported: JPEG, PNG, WebP"},
			})
		case service.ErrImageTooSmall:
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "file", Message: "Image too small. Minimum 50x50 pixels"},
			})
		case service.ErrInvalidImageData:
			return NewValidationError(c, "Validation failed", []ValidationError{
				{Field: "file", Message: "Invalid image data"},
			})
		default:
			log.Error().Err(err).Int32("workspace_id", workspaceID).Msg("Failed to upload image")
			return NewInternalError(c, "Failed to upload image")
		}
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("image_id", metadata.ID).
		Str("entity_type", entityType).
		Msg("Image uploaded successfully")

	return c.JSON(http.StatusCreated, UploadImageResponse{
		ID:           metadata.ID,
		ThumbnailURL: metadata.ThumbnailURL,
		DisplayURL:   metadata.DisplayURL,
		OriginalURL:  metadata.OriginalURL,
	})
}

// DeleteImage handles DELETE /api/v1/images
func (h *ImageHandler) DeleteImage(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Workspace required")
	}

	if h.imageService == nil || !h.imageService.IsEnabled() {
		return NewServiceUnavailableError(c, "Image deletion is disabled (storage not configured)")
	}

	imageURL := c.QueryParam("url")
	if imageURL == "" {
		return NewValidationError(c, "Image URL required", []ValidationError{
			{Field: "url", Message: "URL is required"},
		})
	}

	// Verify the image URL belongs to the user's workspace
	// URL format: http://endpoint/bucket/{workspace_id}/...
	expectedPrefix := fmt.Sprintf("/%d/", workspaceID)
	if !strings.Contains(imageURL, expectedPrefix) {
		log.Warn().
			Int32("workspace_id", workspaceID).
			Str("url", imageURL).
			Msg("Attempted to delete image from different workspace")
		return NewForbiddenError(c, "Cannot delete images from another workspace")
	}

	if err := h.imageService.DeleteAllVariants(c.Request().Context(), imageURL); err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Str("url", imageURL).Msg("Failed to delete image")
		return NewInternalError(c, "Failed to delete image")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("url", imageURL).
		Msg("Image deleted successfully")

	return c.NoContent(http.StatusNoContent)
}
