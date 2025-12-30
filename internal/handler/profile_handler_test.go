package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestGetProfile_Success(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	// Pre-create user
	auth0ID := "auth0|profile123"
	name := "Test User"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "test@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, auth0ID, "test@example.com", name, "")

	err := handler.GetProfile(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response ProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", response.Email)
	}

	if *response.Name != name {
		t.Errorf("Expected name '%s', got '%s'", name, *response.Name)
	}
}

func TestGetProfile_MissingAuth0ID(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set up auth context

	err := handler.GetProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetProfile_UserNotFound(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, "auth0|nonexistent", "test@example.com", "Test", "")

	err := handler.GetProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	// Pre-create user
	auth0ID := "auth0|update123"
	oldName := "Old Name"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "update@example.com",
		Name:    &oldName,
	}
	userRepo.AddUser(existingUser)

	reqBody := `{"name": "New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, auth0ID, "update@example.com", oldName, "")

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response ProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if *response.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", *response.Name)
	}

	// Email should remain unchanged
	if response.Email != "update@example.com" {
		t.Errorf("Expected email to remain 'update@example.com', got %s", response.Email)
	}
}

func TestUpdateProfile_MissingAuth0ID(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	reqBody := `{"name": "New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set up auth context

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestUpdateProfile_EmptyName(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	reqBody := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, "auth0|test", "test@example.com", "Test", "")

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if problemDetails.Type != ErrorTypeValidation {
		t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, problemDetails.Type)
	}
}

func TestUpdateProfile_WhitespaceOnlyName(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	reqBody := `{"name": "   "}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, "auth0|test", "test@example.com", "Test", "")

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	reqBody := `{"name": "New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, "auth0|nonexistent", "test@example.com", "Test", "")

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateProfile_NameTooLong(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	profileService := service.NewProfileService(userRepo)
	handler := NewProfileHandler(profileService)

	// Create a name that exceeds 255 characters
	longName := strings.Repeat("a", 256)
	reqBody := `{"name": "` + longName + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, "auth0|test", "test@example.com", "Test", "")

	err := handler.UpdateProfile(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if problemDetails.Type != ErrorTypeValidation {
		t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, problemDetails.Type)
	}
}
