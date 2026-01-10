package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Frequency string

const (
	FrequencyMonthly Frequency = "monthly"
	// Future: FrequencyWeekly, FrequencyYearly
)

// RecurringTransaction represents a recurring transaction template
// Note: The table is still named recurring_transactions for backward compatibility
type RecurringTransaction struct {
	ID          int32           `json:"id"`
	WorkspaceID int32           `json:"workspaceId"`
	Name        string          `json:"name"`
	Amount      decimal.Decimal `json:"amount"`
	AccountID   int32           `json:"accountId"`
	Type        TransactionType `json:"type"`
	CategoryID  *int32          `json:"categoryId,omitempty"`
	Frequency   Frequency       `json:"frequency"`
	DueDay      int32           `json:"dueDay"` // Day of month for projections

	// V2 Fields - Date-based projection control
	StartDate time.Time  `json:"startDate"`          // When the recurring pattern starts
	EndDate   *time.Time `json:"endDate,omitempty"`  // Optional end date (NULL = runs forever)

	IsActive  bool       `json:"isActive"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// RecurringTemplateWithProjectionRange includes projection info
type RecurringTemplateWithProjectionRange struct {
	RecurringTransaction
	FirstProjectionDate *time.Time `json:"firstProjectionDate,omitempty"`
	LastProjectionDate  *time.Time `json:"lastProjectionDate,omitempty"`
}

// CreateRecurringRequest represents a request to create a recurring template
type CreateRecurringRequest struct {
	Name       string          `json:"name" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required,gt=0"`
	AccountID  int32           `json:"accountId" validate:"required"`
	Type       TransactionType `json:"type" validate:"required,oneof=income expense"`
	CategoryID *int32          `json:"categoryId,omitempty"`
	Frequency  Frequency       `json:"frequency" validate:"required,oneof=monthly"`
	DueDay     int32           `json:"dueDay" validate:"required,min=1,max=31"`
	StartDate  time.Time       `json:"startDate" validate:"required"`
	EndDate    *time.Time      `json:"endDate,omitempty"`
}

// UpdateRecurringRequest represents a request to update a recurring template
type UpdateRecurringRequest struct {
	Name       string          `json:"name" validate:"required"`
	Amount     decimal.Decimal `json:"amount" validate:"required,gt=0"`
	AccountID  int32           `json:"accountId" validate:"required"`
	Type       TransactionType `json:"type" validate:"required,oneof=income expense"`
	CategoryID *int32          `json:"categoryId,omitempty"`
	Frequency  Frequency       `json:"frequency" validate:"required,oneof=monthly"`
	DueDay     int32           `json:"dueDay" validate:"required,min=1,max=31"`
	StartDate  time.Time       `json:"startDate" validate:"required"`
	EndDate    *time.Time      `json:"endDate,omitempty"`
	IsActive   bool            `json:"isActive"`
}

type RecurringRepository interface {
	Create(rt *RecurringTransaction) (*RecurringTransaction, error)
	GetByID(workspaceID int32, id int32) (*RecurringTransaction, error)
	ListByWorkspace(workspaceID int32, activeOnly *bool) ([]*RecurringTransaction, error)
	Update(rt *RecurringTransaction) (*RecurringTransaction, error)
	Delete(workspaceID int32, id int32) error
	CheckTransactionExists(recurringID, workspaceID int32, year, month int) (bool, error)

	// V2 Methods - to be implemented in Epic 2
	// GetActiveTemplates(workspaceID int32) ([]*RecurringTransaction, error)
	// GetTemplateWithProjectionRange(workspaceID int32, id int32) (*RecurringTemplateWithProjectionRange, error)
	// SetEndDate(workspaceID int32, id int32, endDate *time.Time) (*RecurringTransaction, error)
	// ToggleActive(workspaceID int32, id int32) (*RecurringTransaction, error)
}
