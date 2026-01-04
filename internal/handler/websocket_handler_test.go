package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// mockJWTValidator is a test double for JWT validation
type mockJWTValidator struct {
	workspaceID int32
	err         error
}

func (m *mockJWTValidator) ValidateToken(token string) (workspaceID int32, err error) {
	return m.workspaceID, m.err
}

var testAllowedOrigins = []string{"http://localhost:3000", "https://fortuna.app"}

func TestWebSocketHandler_HandleWS_MissingToken(t *testing.T) {
	e := echo.New()
	hub := websocket.NewHub()
	validator := &mockJWTValidator{workspaceID: 1, err: nil}
	h := NewWebSocketHandler(hub, validator, testAllowedOrigins)

	// Request without token
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.HandleWS(c)

	// Should return 401 for missing token
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

func TestWebSocketHandler_HandleWS_InvalidToken(t *testing.T) {
	e := echo.New()
	hub := websocket.NewHub()
	validator := &mockJWTValidator{workspaceID: 0, err: echo.NewHTTPError(http.StatusUnauthorized, "invalid token")}
	h := NewWebSocketHandler(hub, validator, testAllowedOrigins)

	// Request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/ws?token=invalid-jwt", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.HandleWS(c)

	// Should return 401 for invalid token
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

func TestWebSocketHandler_HandleWS_ValidToken_NoUpgrade(t *testing.T) {
	e := echo.New()
	hub := websocket.NewHub()
	validator := &mockJWTValidator{workspaceID: 42, err: nil}
	h := NewWebSocketHandler(hub, validator, testAllowedOrigins)

	// Request with valid token but not a WebSocket upgrade request
	req := httptest.NewRequest(http.MethodGet, "/ws?token=valid-jwt", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.HandleWS(c)

	// gorilla/websocket returns an error when upgrade fails (no upgrade headers)
	// This is expected behavior - we're testing auth passes first
	assert.Error(t, err)
	// The error should be about upgrade failure, not auth
	assert.NotContains(t, err.Error(), "unauthorized")
}

func TestWebSocketHandler_CheckOrigin(t *testing.T) {
	hub := websocket.NewHub()
	validator := &mockJWTValidator{workspaceID: 1, err: nil}
	h := NewWebSocketHandler(hub, validator, testAllowedOrigins)

	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{"allowed origin", "http://localhost:3000", true},
		{"allowed origin https", "https://fortuna.app", true},
		{"disallowed origin", "https://evil.com", false},
		{"empty origin (same-origin)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			result := h.checkOrigin(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}
