package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// problemDetails represents an RFC 7807 Problem Details response
type problemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// Error types
const (
	errorTypeUnauthorized = "https://fortuna.app/errors/unauthorized"
)

// unauthorizedError creates an unauthorized error response
func unauthorizedError(c echo.Context, detail string) error {
	return c.JSON(http.StatusUnauthorized, problemDetails{
		Type:     errorTypeUnauthorized,
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}
