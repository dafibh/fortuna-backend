package handler

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/labstack/echo/v4"
)

// MockImageRepository implements storage.ImageRepository for testing
type MockImageRepository struct {
	uploadFunc             func(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error)
	deleteFunc             func(ctx context.Context, objectPath string) error
	generatePresignedURL   func(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
}

func (m *MockImageRepository) Upload(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, objectPath, data, contentType, size)
	}
	return objectPath, nil
}

func (m *MockImageRepository) Delete(ctx context.Context, objectPath string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, objectPath)
	}
	return nil
}

func (m *MockImageRepository) GeneratePresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	if m.generatePresignedURL != nil {
		return m.generatePresignedURL(ctx, objectPath, expiry)
	}
	return "https://s3.amazonaws.com/bucket/" + objectPath + "?X-Amz-Signature=test", nil
}

// createTestImageData creates a valid JPEG image for testing
func createTestImageData(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

// createMultipartForm creates a multipart form with file data
func createMultipartForm(fieldName, filename string, data []byte, entityType string) (*bytes.Buffer, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile(fieldName, filename)
	part.Write(data)

	if entityType != "" {
		writer.WriteField("entityType", entityType)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

// setWorkspaceInContext sets the workspace ID in the request context
func setWorkspaceInContext(c echo.Context, workspaceID int32) {
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, workspaceID)
	c.SetRequest(c.Request().WithContext(ctx))
}

func TestUploadImage_Success(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set workspace ID in request context
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestUploadImage_NoWorkspace(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set workspace ID - should fail
	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUploadImage_NoFile(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUploadImage_InvalidEntityType(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "invalid")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUploadImage_FileTooLarge(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	// Create a file larger than 5MB (just the header, actual validation happens in service)
	largeData := make([]byte, 6*1024*1024)
	body, contentType := createMultipartForm("file", "test.jpg", largeData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUploadImage_InvalidFormat(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.gif", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUploadImage_UploadError(t *testing.T) {
	mockRepo := &MockImageRepository{
		uploadFunc: func(ctx context.Context, objectPath string, data io.Reader, contentType string, size int64) (string, error) {
			return "", errors.New("upload failed")
		},
	}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUploadImage_StorageNotConfigured(t *testing.T) {
	imageSvc := service.NewImageService(nil)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestDeleteImage_Success(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images?path=1/notes/0/test_display.jpg", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.DeleteImage(c)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestDeleteImage_NoWorkspace(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images?path=1/notes/0/test_display.jpg", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DeleteImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestDeleteImage_NoPath(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.DeleteImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestDeleteImage_WrongWorkspace(t *testing.T) {
	mockRepo := &MockImageRepository{}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	e := echo.New()
	// Path contains workspace ID 2, but user is in workspace 1
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/images?path=2/notes/0/test_display.jpg", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1) // User is in workspace 1, but path is for workspace 2

	err := handler.DeleteImage(c)
	if err != nil {
		t.Errorf("expected error response, got error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUploadImage_PresignedURLError(t *testing.T) {
	// Test that presigned URL errors don't fail the upload but are handled gracefully
	mockRepo := &MockImageRepository{
		generatePresignedURL: func(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
			// Verify 2-hour expiry is being passed
			expectedExpiry := 2 * time.Hour
			if expiry != expectedExpiry {
				t.Errorf("expected expiry %v, got %v", expectedExpiry, expiry)
			}
			return "", errors.New("presigned URL generation failed")
		},
	}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected no error (graceful handling), got %v", err)
	}

	// Upload should still succeed even if presigned URL generation fails
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestPresignedURL_ExpiryValidation(t *testing.T) {
	// Test that the correct 2-hour expiry is passed to the repository
	var capturedExpiry time.Duration
	mockRepo := &MockImageRepository{
		generatePresignedURL: func(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
			capturedExpiry = expiry
			return "https://example.com/presigned", nil
		},
	}
	imageSvc := service.NewImageService(mockRepo)
	handler := NewImageHandler(imageSvc)

	imageData := createTestImageData(100, 100)
	body, contentType := createMultipartForm("file", "test.jpg", imageData, "notes")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setWorkspaceInContext(c, 1)

	err := handler.UploadImage(c)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedExpiry := 2 * time.Hour
	if capturedExpiry != expectedExpiry {
		t.Errorf("expected expiry %v, got %v", expectedExpiry, capturedExpiry)
	}
}
