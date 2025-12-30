package domain

import "errors"

// Domain errors
var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInternalError     = errors.New("internal error")
	ErrUserNotFound      = errors.New("user not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrAccountNotFound   = errors.New("account not found")
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name exceeds maximum length")
	ErrInvalidTemplate   = errors.New("invalid template")
)

// Validation constants
const (
	MaxAccountNameLength = 255
)
