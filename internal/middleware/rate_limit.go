package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	// DefaultRateLimit is the default rate limit per minute
	DefaultRateLimit = 100
	// DefaultBurstSize is the default burst size
	DefaultBurstSize = 10
	// CleanupInterval is the interval for cleaning up stale limiters
	CleanupInterval = 5 * time.Minute
	// LimiterTTL is the time-to-live for inactive limiters
	LimiterTTL = 10 * time.Minute
)

// RateLimiter manages per-token rate limiting
type RateLimiter struct {
	limiters  map[uuid.UUID]*limiterEntry
	mu        sync.RWMutex
	rateLimit float64
	burstSize int
	stopCh    chan struct{}
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new RateLimiter with default settings
func NewRateLimiter() *RateLimiter {
	return NewRateLimiterWithConfig(DefaultRateLimit, DefaultBurstSize)
}

// NewRateLimiterWithConfig creates a RateLimiter with custom configuration
func NewRateLimiterWithConfig(requestsPerMinute int, burstSize int) *RateLimiter {
	rl := &RateLimiter{
		limiters:  make(map[uuid.UUID]*limiterEntry),
		rateLimit: float64(requestsPerMinute) / 60.0, // Convert to per-second
		burstSize: burstSize,
		stopCh:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given token is allowed
func (r *RateLimiter) Allow(tokenID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.limiters[tokenID]
	if !exists {
		entry = &limiterEntry{
			limiter:  rate.NewLimiter(rate.Limit(r.rateLimit), r.burstSize),
			lastSeen: time.Now(),
		}
		r.limiters[tokenID] = entry
	} else {
		entry.lastSeen = time.Now()
	}

	return entry.limiter.Allow()
}

// GetState returns the current state for rate limit headers
func (r *RateLimiter) GetState(tokenID uuid.UUID) (remaining int, resetTime time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.limiters[tokenID]
	if !exists {
		return r.burstSize, time.Now().Add(time.Minute)
	}

	// Estimate remaining tokens (approximation)
	tokens := int(entry.limiter.Tokens())
	if tokens < 0 {
		tokens = 0
	}

	// Reset time is approximately when tokens would be fully replenished
	resetDuration := time.Duration(float64(r.burstSize-tokens)/r.rateLimit) * time.Second
	return tokens, time.Now().Add(resetDuration)
}

// cleanup periodically removes stale limiters to prevent memory leaks
func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			now := time.Now()
			for tokenID, entry := range r.limiters {
				if now.Sub(entry.lastSeen) > LimiterTTL {
					delete(r.limiters, tokenID)
					log.Debug().Str("token_id", tokenID.String()).Msg("Cleaned up stale rate limiter")
				}
			}
			r.mu.Unlock()
		case <-r.stopCh:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (r *RateLimiter) Stop() {
	close(r.stopCh)
}

// RateLimitMiddleware returns an Echo middleware that applies rate limiting
func RateLimitMiddleware(rl *RateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only apply rate limiting to API token authenticated requests
			if !IsAPITokenAuth(c) {
				return next(c)
			}

			tokenID := GetAPITokenID(c)
			if tokenID == uuid.Nil {
				// No token ID in context, skip rate limiting
				return next(c)
			}

			// Check rate limit
			if !rl.Allow(tokenID) {
				_, resetTime := rl.GetState(tokenID)
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}

				// Set rate limit headers
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", DefaultRateLimit))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))

				log.Warn().
					Str("token_id", tokenID.String()).
					Int("retry_after", retryAfter).
					Msg("Rate limit exceeded")

				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"type":   "https://fortuna.app/errors/rate-limit",
					"title":  "Rate Limit Exceeded",
					"status": 429,
					"detail": fmt.Sprintf("Too many requests. Please retry after %d seconds.", retryAfter),
				})
			}

			// Add rate limit headers to successful responses
			remaining, resetTime := rl.GetState(tokenID)
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", DefaultRateLimit))
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			c.Response().Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			return next(c)
		}
	}
}
