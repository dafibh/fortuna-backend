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

func TestGetAPITokens_Success(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test", "", workspaceID)

	err := handler.GetAPITokens(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestGetAPITokens_MissingWorkspace(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set workspace ID
	setupAuthContext(c, "auth0|test", "test@example.com", "Test", "")

	err := handler.GetAPITokens(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCreateAPIToken_Success(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	// Pre-create user
	auth0ID := "auth0|token123"
	name := "Test User"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "test@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	workspaceID := int32(1)

	reqBody := `{"description": "Test token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-tokens", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, auth0ID, "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateAPIToken(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response domain.CreateAPITokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Description != "Test token" {
		t.Errorf("Expected description 'Test token', got %s", response.Description)
	}

	if !strings.HasPrefix(response.Token, "fort_") {
		t.Errorf("Expected token to start with 'fort_', got %s", response.Token[:10])
	}
}

func TestCreateAPIToken_MissingDescription(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	// Pre-create user
	auth0ID := "auth0|token456"
	name := "Test User"
	existingUser := &domain.User{
		ID:      uuid.New(),
		Auth0ID: auth0ID,
		Email:   "test@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	workspaceID := int32(1)

	reqBody := `{"description": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-tokens", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, auth0ID, "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateAPIToken(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestRevokeAPIToken_Success(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	// Pre-create user
	auth0ID := "auth0|revoke123"
	name := "Test User"
	userID := uuid.New()
	existingUser := &domain.User{
		ID:      userID,
		Auth0ID: auth0ID,
		Email:   "test@example.com",
		Name:    &name,
	}
	userRepo.AddUser(existingUser)

	workspaceID := int32(1)

	// Create a token first
	token := &domain.APIToken{
		ID:          uuid.New(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		Description: "Test token",
		TokenHash:   "somehash",
		TokenPrefix: "fort_abc...",
	}
	tokenRepo.AddToken(token)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-tokens/"+token.ID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(token.ID.String())

	setupAuthContextWithWorkspace(c, auth0ID, "test@example.com", "Test User", "", workspaceID)

	err := handler.RevokeAPIToken(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestRevokeAPIToken_NotFound(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	workspaceID := int32(1)
	nonExistentID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-tokens/"+nonExistentID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(nonExistentID.String())

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test", "", workspaceID)

	err := handler.RevokeAPIToken(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestRevokeAPIToken_InvalidID(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-tokens/invalid-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid-uuid")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test", "", workspaceID)

	err := handler.RevokeAPIToken(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestRevokeAPIToken_WrongWorkspace(t *testing.T) {
	e := echo.New()
	tokenRepo := testutil.NewMockAPITokenRepository()
	userRepo := testutil.NewMockUserRepository()
	workspaceRepo := testutil.NewMockWorkspaceRepository()
	tokenService := service.NewAPITokenService(tokenRepo)
	authService := service.NewAuthService(userRepo, workspaceRepo)
	handler := NewAPITokenHandler(tokenService, authService)

	// Token belongs to workspace 1
	token := &domain.APIToken{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		WorkspaceID: int32(1),
		Description: "Test token",
		TokenHash:   "somehash",
		TokenPrefix: "fort_abc...",
	}
	tokenRepo.AddToken(token)

	// But we're authenticated with workspace 2
	workspaceID := int32(2)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-tokens/"+token.ID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(token.ID.String())

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test", "", workspaceID)

	err := handler.RevokeAPIToken(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	// Should return 404 (not 403) because token not found in this workspace
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}
