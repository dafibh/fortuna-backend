package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Helper to set up auth context
func setupAuthContext(c echo.Context, auth0ID string, email, name, picture string) {
	setupAuthContextWithWorkspace(c, auth0ID, email, name, picture, 0)
}

// Helper to set up auth context with workspace ID
func setupAuthContextWithWorkspace(c echo.Context, auth0ID string, email, name, picture string, workspaceID int32) {
	customClaims := &middleware.CustomClaims{
		Email:   email,
		Name:    name,
		Picture: picture,
	}
	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Subject: auth0ID,
		},
		CustomClaims: customClaims,
	}
	ctx := context.WithValue(c.Request().Context(), middleware.ClaimsKey, claims)
	ctx = context.WithValue(ctx, middleware.Auth0IDKey, auth0ID)
	if workspaceID > 0 {
		ctx = context.WithValue(ctx, middleware.WorkspaceIDKey, workspaceID)
	}
	c.SetRequest(c.Request().WithContext(ctx))
}

func TestCallback_NewUser(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set up auth context with claims
	setupAuthContext(c, "auth0|newuser123", "new@example.com", "New User", "https://example.com/pic.jpg")

	err := handler.Callback(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response AuthCallbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.IsNewUser {
		t.Error("Expected IsNewUser to be true for new user")
	}

	if response.User.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got %s", response.User.Email)
	}

	if response.Workspace.Name != "Personal" {
		t.Errorf("Expected workspace name 'Personal', got %s", response.Workspace.Name)
	}
}

func TestCallback_ExistingUser(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	// Pre-create user and workspace
	auth0ID := "auth0|existing123"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "existing@example.com",
	}
	userRepo.AddUser(existingUser)

	existingWorkspace := &domain.Workspace{
		ID:     1,
		UserID: existingUser.ID,
		Name:   "My Workspace",
	}
	workspaceRepo.AddWorkspace(existingWorkspace, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContext(c, auth0ID, "existing@example.com", "Existing User", "")

	err := handler.Callback(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response AuthCallbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.IsNewUser {
		t.Error("Expected IsNewUser to be false for existing user")
	}

	if response.Workspace.Name != "My Workspace" {
		t.Errorf("Expected workspace name 'My Workspace', got %s", response.Workspace.Name)
	}
}

func TestCallback_MissingAuth0ID(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set up auth context - simulating missing auth

	err := handler.Callback(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCallback_MissingEmail(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set up auth context WITHOUT email
	setupAuthContext(c, "auth0|noemail123", "", "No Email User", "")

	err := handler.Callback(c)
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

func TestMe_Success(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	// Pre-create user and workspace
	auth0ID := "auth0|me123"
	name := "Me User"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "me@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	existingWorkspace := &domain.Workspace{
		ID:     1,
		UserID: existingUser.ID,
		Name:   "Personal",
	}
	workspaceRepo.AddWorkspace(existingWorkspace, "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Include workspace ID in context (simulating middleware behavior)
	setupAuthContextWithWorkspace(c, auth0ID, "me@example.com", name, "", existingWorkspace.ID)

	err := handler.Me(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response AuthCallbackResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.User.Email != "me@example.com" {
		t.Errorf("Expected email 'me@example.com', got %s", response.User.Email)
	}
}

func TestMe_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Auth context without workspace ID
	setupAuthContext(c, "auth0|noworkspace", "test@example.com", "Test User", "")

	err := handler.Me(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestMe_UserNotFound(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// User doesn't exist in repo but has workspace ID
	setupAuthContextWithWorkspace(c, "auth0|notfound", "notfound@example.com", "Not Found", "", 1)

	err := handler.Me(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestLogout_Success(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set up auth context
	setupAuthContext(c, "auth0|logout123", "logout@example.com", "Logout User", "")

	err := handler.Logout(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LogoutResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Logged out successfully" {
		t.Errorf("Expected message 'Logged out successfully', got %s", response.Message)
	}
}

func TestLogout_MissingAuth0ID(t *testing.T) {
	e := echo.New()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAuthHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set up auth context - simulating missing auth

	err := handler.Logout(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if problemDetails.Type != ErrorTypeUnauthorized {
		t.Errorf("Expected error type %s, got %s", ErrorTypeUnauthorized, problemDetails.Type)
	}

	if problemDetails.Status != http.StatusUnauthorized {
		t.Errorf("Expected status %d in body, got %d", http.StatusUnauthorized, problemDetails.Status)
	}
}
