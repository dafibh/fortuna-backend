package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiterWithConfig(10, 5) // 10 per minute, burst of 5
	defer rl.Stop()

	tokenID := uuid.New()

	// First 5 requests should be allowed (burst)
	for i := 0; i < 5; i++ {
		if !rl.Allow(tokenID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be rate limited (exceeded burst)
	if rl.Allow(tokenID) {
		t.Error("Request 6 should be rate limited")
	}
}

func TestRateLimiter_DifferentTokens(t *testing.T) {
	rl := NewRateLimiterWithConfig(10, 3)
	defer rl.Stop()

	token1 := uuid.New()
	token2 := uuid.New()

	// Exhaust token1's burst
	for i := 0; i < 3; i++ {
		if !rl.Allow(token1) {
			t.Errorf("Token1 request %d should be allowed", i+1)
		}
	}

	// Token1 should be rate limited
	if rl.Allow(token1) {
		t.Error("Token1 should be rate limited")
	}

	// Token2 should still have its full burst
	for i := 0; i < 3; i++ {
		if !rl.Allow(token2) {
			t.Errorf("Token2 request %d should be allowed", i+1)
		}
	}
}

func TestRateLimitMiddleware_SkipsNonAPIToken(t *testing.T) {
	e := echo.New()
	rl := NewRateLimiterWithConfig(1, 1)
	defer rl.Stop()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set IsAPITokenAuthKey - simulating JWT auth
	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "OK")
	}

	// Should pass through without rate limiting
	for i := 0; i < 5; i++ {
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)
		handlerCalled = false

		err := RateLimitMiddleware(rl)(handler)(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !handlerCalled {
			t.Error("Handler should be called for non-API-token requests")
		}
	}
}

func TestRateLimitMiddleware_RateLimitsAPIToken(t *testing.T) {
	e := echo.New()
	rl := NewRateLimiterWithConfig(10, 2) // Small burst for testing
	defer rl.Stop()

	tokenID := uuid.New()

	// Create a context with API token auth markers
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set API token context
	ctx := context.WithValue(req.Context(), IsAPITokenAuthKey, true)
	ctx = context.WithValue(ctx, APITokenIDKey, tokenID)
	c.SetRequest(req.WithContext(ctx))

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	// First 2 requests should succeed (burst)
	for i := 0; i < 2; i++ {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
		ctx = context.WithValue(req.Context(), IsAPITokenAuthKey, true)
		ctx = context.WithValue(ctx, APITokenIDKey, tokenID)
		c = e.NewContext(req.WithContext(ctx), rec)

		err := RateLimitMiddleware(rl)(handler)(c)
		if err != nil {
			t.Fatalf("Request %d: Expected no error, got %v", i+1, err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, rec.Code)
		}
		// Check rate limit headers are present
		if rec.Header().Get("X-RateLimit-Limit") == "" {
			t.Errorf("Request %d: Expected X-RateLimit-Limit header", i+1)
		}
	}

	// 3rd request should be rate limited
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	ctx = context.WithValue(req.Context(), IsAPITokenAuthKey, true)
	ctx = context.WithValue(ctx, APITokenIDKey, tokenID)
	c = e.NewContext(req.WithContext(ctx), rec)

	err := RateLimitMiddleware(rl)(handler)(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header")
	}
}
