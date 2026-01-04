package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MockAPITokenValidator implements APITokenValidator for testing
type MockAPITokenValidator struct {
	token *domain.APIToken
	err   error
}

func (m *MockAPITokenValidator) ValidateToken(ctx context.Context, token string) (*domain.APIToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func TestAPITokenAuth_Success(t *testing.T) {
	e := echo.New()
	tokenID := uuid.New()
	userID := uuid.New()
	workspaceID := int32(1)

	validator := &MockAPITokenValidator{
		token: &domain.APIToken{
			ID:          tokenID,
			UserID:      userID,
			WorkspaceID: workspaceID,
		},
	}

	middleware := NewAPITokenAuthMiddleware(validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer fort_testtoken123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		// Verify context values are set
		if GetWorkspaceID(c) != workspaceID {
			t.Errorf("Expected workspace ID %d, got %d", workspaceID, GetWorkspaceID(c))
		}
		if GetUserID(c) != userID {
			t.Errorf("Expected user ID %s, got %s", userID, GetUserID(c))
		}
		if GetAPITokenID(c) != tokenID {
			t.Errorf("Expected token ID %s, got %s", tokenID, GetAPITokenID(c))
		}
		if !IsAPITokenAuth(c) {
			t.Error("Expected IsAPITokenAuth to be true")
		}
		return c.String(http.StatusOK, "OK")
	}

	err := middleware.Authenticate()(handler)(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("Handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestAPITokenAuth_MissingHeader(t *testing.T) {
	e := echo.New()

	validator := &MockAPITokenValidator{}
	middleware := NewAPITokenAuthMiddleware(validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	// No Authorization header
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := middleware.Authenticate()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestAPITokenAuth_InvalidFormat(t *testing.T) {
	e := echo.New()

	validator := &MockAPITokenValidator{}
	middleware := NewAPITokenAuthMiddleware(validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Invalid format")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := middleware.Authenticate()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestAPITokenAuth_NotFortToken(t *testing.T) {
	e := echo.New()

	validator := &MockAPITokenValidator{}
	middleware := NewAPITokenAuthMiddleware(validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer jwt_token_here")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := middleware.Authenticate()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestAPITokenAuth_InvalidToken(t *testing.T) {
	e := echo.New()

	validator := &MockAPITokenValidator{
		err: domain.ErrAPITokenNotFound,
	}
	middleware := NewAPITokenAuthMiddleware(validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer fort_invalidtoken")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := middleware.Authenticate()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}
