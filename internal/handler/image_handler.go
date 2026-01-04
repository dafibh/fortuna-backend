package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	ID            string `json:"id"`
	ThumbnailPath string `json:"thumbnailPath"`
	ThumbnailURL  string `json:"thumbnailUrl"`
	DisplayPath   string `json:"displayPath"`
	DisplayURL    string `json:"displayUrl"`
	OriginalPath  string `json:"originalPath"`
	OriginalURL   string `json:"originalUrl"`
}

// PresignedURLResponse represents a single presigned URL response
type PresignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expiresAt"`
}

// BatchPresignedURLRequest represents the batch presigned URL request body
type BatchPresignedURLRequest struct {
	Paths []string `json:"paths"`
}

// BatchPresignedURLItem represents a single item in the batch response
type BatchPresignedURLItem struct {
	Path      string  `json:"path"`
	URL       *string `json:"url"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// BatchPresignedURLResponse represents the batch presigned URL response
type BatchPresignedURLResponse struct {
	URLs []BatchPresignedURLItem `json:"urls"`
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

	// Generate presigned URLs for immediate use
	ctx := c.Request().Context()
	thumbnailURL, err := h.imageService.GeneratePresignedURL(ctx, metadata.ThumbnailPath)
	if err != nil {
		log.Warn().Err(err).Str("path", metadata.ThumbnailPath).Msg("Failed to generate thumbnail presigned URL")
	}
	displayURL, err := h.imageService.GeneratePresignedURL(ctx, metadata.DisplayPath)
	if err != nil {
		log.Warn().Err(err).Str("path", metadata.DisplayPath).Msg("Failed to generate display presigned URL")
	}
	originalURL, err := h.imageService.GeneratePresignedURL(ctx, metadata.OriginalPath)
	if err != nil {
		log.Warn().Err(err).Str("path", metadata.OriginalPath).Msg("Failed to generate original presigned URL")
	}

	return c.JSON(http.StatusCreated, UploadImageResponse{
		ID:            metadata.ID,
		ThumbnailPath: metadata.ThumbnailPath,
		ThumbnailURL:  thumbnailURL,
		DisplayPath:   metadata.DisplayPath,
		DisplayURL:    displayURL,
		OriginalPath:  metadata.OriginalPath,
		OriginalURL:   originalURL,
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

	// Accept either path (new) or url (legacy) parameter
	objectPath := c.QueryParam("path")
	if objectPath == "" {
		objectPath = c.QueryParam("url") // Legacy support
	}
	if objectPath == "" {
		return NewValidationError(c, "Image path required", []ValidationError{
			{Field: "path", Message: "Path is required"},
		})
	}

	// Verify the object path belongs to the user's workspace
	// Path format: {workspace_id}/{entity_type}/{entity_id}/{image_id}_{variant}.jpg
	expectedPrefix := fmt.Sprintf("%d/", workspaceID)
	if !strings.HasPrefix(objectPath, expectedPrefix) {
		log.Warn().
			Int32("workspace_id", workspaceID).
			Str("path", objectPath).
			Msg("Attempted to delete image from different workspace")
		return NewForbiddenError(c, "Cannot delete images from another workspace")
	}

	if err := h.imageService.DeleteAllVariants(c.Request().Context(), objectPath); err != nil {
		log.Error().Err(err).Int32("workspace_id", workspaceID).Str("path", objectPath).Msg("Failed to delete image")
		return NewInternalError(c, "Failed to delete image")
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("path", objectPath).
		Msg("Image deleted successfully")

	return c.NoContent(http.StatusNoContent)
}

// GetPresignedURL handles GET /api/v1/images/url
// @Summary Get presigned URL for an image
// @Description Generate a presigned URL for temporary access to a private S3 image
// @Tags images
// @Accept json
// @Produce json
// @Param path query string true "Object path (e.g., 1/wishlist_items/5/abc_display.jpg)"
// @Success 200 {object} PresignedURLResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 404 {object} ProblemDetails
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /images/url [get]
func (h *ImageHandler) GetPresignedURL(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Authentication required")
	}

	if h.imageService == nil || !h.imageService.IsEnabled() {
		return NewServiceUnavailableError(c, "Image service is disabled (storage not configured)")
	}

	// Get object path from query parameter
	objectPath := c.QueryParam("path")
	if objectPath == "" {
		return NewValidationError(c, "Path is required", []ValidationError{
			{Field: "path", Message: "Path query parameter is required"},
		})
	}

	// Validate path format and extract workspace ID from path
	// Path format: {workspace_id}/{entity_type}/{entity_id}/{image_id}_{variant}.{ext}
	pathParts := strings.SplitN(objectPath, "/", 2)
	if len(pathParts) < 2 {
		return NewNotFoundError(c, "Image not found")
	}

	pathWorkspaceID, err := strconv.ParseInt(pathParts[0], 10, 32)
	if err != nil {
		return NewNotFoundError(c, "Image not found")
	}

	// Verify ownership - return 404 to prevent enumeration
	if int32(pathWorkspaceID) != workspaceID {
		log.Warn().
			Int32("workspace_id", workspaceID).
			Int64("path_workspace_id", pathWorkspaceID).
			Str("path", objectPath).
			Msg("Attempted to access image from different workspace")
		return NewNotFoundError(c, "Image not found")
	}

	// Generate presigned URL
	url, err := h.imageService.GeneratePresignedURL(c.Request().Context(), objectPath)
	if err != nil {
		log.Error().Err(err).Str("path", objectPath).Msg("Failed to generate presigned URL")
		return NewNotFoundError(c, "Image not found")
	}

	expiresAt := time.Now().Add(2 * time.Hour).Format(time.RFC3339)

	return c.JSON(http.StatusOK, PresignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt,
	})
}

// GetBatchPresignedURLs handles POST /api/v1/images/urls
// @Summary Get presigned URLs for multiple images
// @Description Generate presigned URLs for multiple private S3 images in a single request
// @Tags images
// @Accept json
// @Produce json
// @Param request body BatchPresignedURLRequest true "Array of object paths"
// @Success 200 {object} BatchPresignedURLResponse
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /images/urls [post]
func (h *ImageHandler) GetBatchPresignedURLs(c echo.Context) error {
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == 0 {
		return NewUnauthorizedError(c, "Authentication required")
	}

	if h.imageService == nil || !h.imageService.IsEnabled() {
		return NewServiceUnavailableError(c, "Image service is disabled (storage not configured)")
	}

	var req BatchPresignedURLRequest
	if err := c.Bind(&req); err != nil {
		return NewValidationError(c, "Invalid request body", nil)
	}

	if len(req.Paths) == 0 {
		return NewValidationError(c, "At least one path is required", []ValidationError{
			{Field: "paths", Message: "Paths array cannot be empty"},
		})
	}

	// Limit batch size to prevent abuse
	const maxBatchSize = 50
	if len(req.Paths) > maxBatchSize {
		return NewValidationError(c, "Too many paths", []ValidationError{
			{Field: "paths", Message: fmt.Sprintf("Maximum %d paths per request", maxBatchSize)},
		})
	}

	ctx := c.Request().Context()
	expiresAt := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	results := make([]BatchPresignedURLItem, len(req.Paths))

	for i, objectPath := range req.Paths {
		item := BatchPresignedURLItem{Path: objectPath}

		// Validate path format and extract workspace ID
		pathParts := strings.SplitN(objectPath, "/", 2)
		if len(pathParts) < 2 {
			errMsg := "Invalid path format"
			item.Error = &errMsg
			results[i] = item
			continue
		}

		pathWorkspaceID, err := strconv.ParseInt(pathParts[0], 10, 32)
		if err != nil {
			errMsg := "Invalid path format"
			item.Error = &errMsg
			results[i] = item
			continue
		}

		// Verify ownership - silently skip (with null URL) for security
		if int32(pathWorkspaceID) != workspaceID {
			log.Warn().
				Int32("workspace_id", workspaceID).
				Int64("path_workspace_id", pathWorkspaceID).
				Str("path", objectPath).
				Msg("Batch request included path from different workspace")
			errMsg := "Image not found"
			item.Error = &errMsg
			results[i] = item
			continue
		}

		// Generate presigned URL
		url, err := h.imageService.GeneratePresignedURL(ctx, objectPath)
		if err != nil {
			log.Error().Err(err).Str("path", objectPath).Msg("Failed to generate presigned URL in batch")
			errMsg := "Failed to generate URL"
			item.Error = &errMsg
			results[i] = item
			continue
		}

		item.URL = &url
		item.ExpiresAt = &expiresAt
		results[i] = item
	}

	return c.JSON(http.StatusOK, BatchPresignedURLResponse{URLs: results})
}
