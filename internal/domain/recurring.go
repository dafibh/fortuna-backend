package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// RecurringTemplate represents a recurring template for generating projected transactions
type RecurringTemplate struct {
	ID               int32             `json:"id"`
	WorkspaceID      int32             `json:"workspaceId"`
	Description      string            `json:"description"`
	Amount           decimal.Decimal   `json:"amount"`
	CategoryID       *int32            `json:"categoryId"`       // Optional category
	AccountID        int32             `json:"accountId"`
	Frequency        string            `json:"frequency"`        // 'monthly' for MVP
	StartDate        time.Time         `json:"startDate"`
	EndDate          *time.Time        `json:"endDate"`          // NULL means runs forever
	Notes            *string           `json:"notes"`            // Optional notes for generated transactions
	SettlementIntent *SettlementIntent `json:"settlementIntent"` // For CC accounts: 'immediate' or 'deferred'
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// CreateRecurringTemplateInput represents input for creating a recurring template
type CreateRecurringTemplateInput struct {
	WorkspaceID       int32
	Description       string
	Amount            decimal.Decimal
	CategoryID        *int32            // Optional category
	AccountID         int32
	Frequency         string
	StartDate         time.Time
	EndDate           *time.Time
	Notes             *string           // Optional notes for generated transactions
	SettlementIntent  *SettlementIntent // For CC accounts: 'immediate' or 'deferred'
	LinkTransactionID *int32            // Optional: link an existing transaction to this template
}

// UpdateRecurringTemplateInput represents input for updating a recurring template
type UpdateRecurringTemplateInput struct {
	Description      string
	Amount           decimal.Decimal
	CategoryID       *int32            // Optional category
	AccountID        int32
	Frequency        string
	StartDate        time.Time
	EndDate          *time.Time
	Notes            *string           // Optional notes for generated transactions
	SettlementIntent *SettlementIntent // For CC accounts: 'immediate' or 'deferred'
}

// RecurringTemplateRepository defines the interface for recurring template persistence
type RecurringTemplateRepository interface {
	Create(template *RecurringTemplate) (*RecurringTemplate, error)
	Update(workspaceID int32, id int32, input *UpdateRecurringTemplateInput) (*RecurringTemplate, error)
	Delete(workspaceID int32, id int32) error
	GetByID(workspaceID int32, id int32) (*RecurringTemplate, error)
	ListByWorkspace(workspaceID int32) ([]*RecurringTemplate, error)
	GetActive(workspaceID int32) ([]*RecurringTemplate, error)
	GetAllActive() ([]*RecurringTemplate, error) // For daily sync goroutine
}

// RecurringTemplateService defines the interface for recurring template business logic
type RecurringTemplateService interface {
	CreateTemplate(workspaceID int32, input CreateRecurringTemplateInput) (*RecurringTemplate, error)
	UpdateTemplate(workspaceID int32, id int32, input UpdateRecurringTemplateInput) (*RecurringTemplate, error)
	DeleteTemplate(workspaceID int32, id int32) error
	GetTemplate(workspaceID int32, id int32) (*RecurringTemplate, error)
	ListTemplates(workspaceID int32) ([]*RecurringTemplate, error)
}
