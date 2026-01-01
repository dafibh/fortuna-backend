package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ProblemDetails represents an RFC 7807 Problem Details response
type ProblemDetails struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Status   int               `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	Errors   []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error types
const (
	ErrorTypeValidation   = "https://fortuna.app/errors/validation"
	ErrorTypeNotFound     = "https://fortuna.app/errors/not-found"
	ErrorTypeUnauthorized = "https://fortuna.app/errors/unauthorized"
	ErrorTypeForbidden    = "https://fortuna.app/errors/forbidden"
	ErrorTypeConflict     = "https://fortuna.app/errors/conflict"
	ErrorTypeInternal     = "https://fortuna.app/errors/internal"
)

// NewValidationError creates a validation error response
func NewValidationError(c echo.Context, detail string, errors []ValidationError) error {
	return c.JSON(http.StatusBadRequest, ProblemDetails{
		Type:     ErrorTypeValidation,
		Title:    "Validation Error",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: c.Request().URL.Path,
		Errors:   errors,
	})
}

// NewNotFoundError creates a not found error response
func NewNotFoundError(c echo.Context, detail string) error {
	return c.JSON(http.StatusNotFound, ProblemDetails{
		Type:     ErrorTypeNotFound,
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}

// NewUnauthorizedError creates an unauthorized error response
func NewUnauthorizedError(c echo.Context, detail string) error {
	return c.JSON(http.StatusUnauthorized, ProblemDetails{
		Type:     ErrorTypeUnauthorized,
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}

// NewForbiddenError creates a forbidden error response
func NewForbiddenError(c echo.Context, detail string) error {
	return c.JSON(http.StatusForbidden, ProblemDetails{
		Type:     ErrorTypeForbidden,
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}

// NewConflictError creates a conflict error response
func NewConflictError(c echo.Context, detail string) error {
	return c.JSON(http.StatusConflict, ProblemDetails{
		Type:     ErrorTypeConflict,
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}

// NewInternalError creates an internal error response
func NewInternalError(c echo.Context, detail string) error {
	return c.JSON(http.StatusInternalServerError, ProblemDetails{
		Type:     ErrorTypeInternal,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: c.Request().URL.Path,
	})
}
