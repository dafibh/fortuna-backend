package domain

import "time"

type BudgetCategory struct {
	ID          int32      `json:"id"`
	WorkspaceID int32      `json:"workspaceId"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type BudgetCategoryRepository interface {
	Create(category *BudgetCategory) (*BudgetCategory, error)
	GetByID(workspaceID int32, id int32) (*BudgetCategory, error)
	GetByName(workspaceID int32, name string) (*BudgetCategory, error)
	GetAllByWorkspace(workspaceID int32) ([]*BudgetCategory, error)
	Update(workspaceID int32, id int32, name string) (*BudgetCategory, error)
	SoftDelete(workspaceID int32, id int32) error
	HasTransactions(workspaceID int32, id int32) (bool, error)
}
