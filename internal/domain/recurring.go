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

type RecurringTransaction struct {
	ID          int32           `json:"id"`
	WorkspaceID int32           `json:"workspaceId"`
	Name        string          `json:"name"`
	Amount      decimal.Decimal `json:"amount"`
	AccountID   int32           `json:"accountId"`
	Type        TransactionType `json:"type"`
	CategoryID  *int32          `json:"categoryId,omitempty"`
	Frequency   Frequency       `json:"frequency"`
	DueDay      int32           `json:"dueDay"`
	IsActive    bool            `json:"isActive"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty"`
}

type RecurringRepository interface {
	Create(rt *RecurringTransaction) (*RecurringTransaction, error)
	GetByID(workspaceID int32, id int32) (*RecurringTransaction, error)
	ListByWorkspace(workspaceID int32, activeOnly *bool) ([]*RecurringTransaction, error)
	Update(rt *RecurringTransaction) (*RecurringTransaction, error)
	Delete(workspaceID int32, id int32) error
	CheckTransactionExists(recurringID, workspaceID int32, year, month int) (bool, error)
}
