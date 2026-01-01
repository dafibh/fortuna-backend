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
	ErrNameTooLong            = errors.New("name exceeds maximum length")
	ErrInvalidTemplate        = errors.New("invalid template")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrInvalidTransactionType       = errors.New("invalid transaction type")
	ErrInvalidAmount                = errors.New("amount must be positive")
	ErrNotesTooLong                 = errors.New("notes exceed maximum length")
	ErrInvalidSettlementIntent      = errors.New("invalid settlement intent")
	ErrSettlementIntentNotApplicable = errors.New("settlement intent only applies to credit card transactions")
	ErrTransactionAlreadyPaid       = errors.New("cannot change settlement intent for paid transactions")
	ErrSameAccountTransfer          = errors.New("cannot transfer to the same account")
	ErrMonthNotFound                = errors.New("month not found")
	ErrMonthAlreadyExists           = errors.New("month already exists")
	ErrBudgetCategoryNotFound       = errors.New("budget category not found")
	ErrBudgetCategoryAlreadyExists  = errors.New("budget category with this name already exists")
	ErrBudgetAllocationNotFound     = errors.New("budget allocation not found")
	ErrInvalidAccountType           = errors.New("invalid account type for this operation")
	ErrInvalidSourceAccount         = errors.New("cannot use a credit card as source account for CC payment")
)

// Validation constants
const (
	MaxAccountNameLength        = 255
	MaxTransactionNameLength    = 255
	MaxTransactionNotesLength   = 1000
	MaxBudgetCategoryNameLength = 100
)
