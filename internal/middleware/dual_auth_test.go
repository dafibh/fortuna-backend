package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// Note: These tests use mock middleware functions since we can't easily
// mock the full JWT validator without a lot of setup

func TestDualAuth_JWTOnly_RejectsAPIToken(t *testing.T) {
	e := echo.New()

	// Create a minimal DualAuthMiddleware - JWT auth will fail but we're testing rejection logic
	dualAuth := &DualAuthMiddleware{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil)
	req.Header.Set("Authorization", "Bearer fort_testtoken123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := dualAuth.JWTOnly()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestDualAuth_APITokenOnly_RejectsJWT(t *testing.T) {
	e := echo.New()

	// Create a minimal DualAuthMiddleware
	dualAuth := &DualAuthMiddleware{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/external", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		t.Error("Handler should not be called")
		return nil
	}

	err := dualAuth.APITokenOnly()(handler)(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestDualAuth_MissingHeader(t *testing.T) {
	e := echo.New()

	dualAuth := &DualAuthMiddleware{}

	tests := []struct {
		name       string
		middleware echo.MiddlewareFunc
	}{
		{"Authenticate", dualAuth.Authenticate()},
		{"JWTOnly", dualAuth.JWTOnly()},
		{"APITokenOnly", dualAuth.APITokenOnly()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			// No Authorization header
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := func(c echo.Context) error {
				t.Error("Handler should not be called")
				return nil
			}

			err := tt.middleware(handler)(c)
			if err != nil {
				t.Fatalf("Expected JSON response, got error: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", rec.Code)
			}
		})
	}
}

func TestDualAuth_InvalidHeaderFormat(t *testing.T) {
	e := echo.New()

	dualAuth := &DualAuthMiddleware{}

	tests := []struct {
		name       string
		header     string
		middleware echo.MiddlewareFunc
	}{
		{"Authenticate - no space", "BearerToken", dualAuth.Authenticate()},
		{"JWTOnly - Basic auth", "Basic dXNlcjpwYXNz", dualAuth.JWTOnly()},
		{"APITokenOnly - empty", "", dualAuth.APITokenOnly()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := func(c echo.Context) error {
				t.Error("Handler should not be called")
				return nil
			}

			err := tt.middleware(handler)(c)
			if err != nil {
				t.Fatalf("Expected JSON response, got error: %v", err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", rec.Code)
			}
		})
	}
}
