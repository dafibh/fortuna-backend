package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/shopspring/decimal"
)

var monthFormatRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

var (
	ErrGroupNotFound          = errors.New("transaction group not found")
	ErrGroupNameEmpty         = errors.New("group name cannot be empty")
	ErrInvalidMonthFormat     = errors.New("month must be in YYYY-MM format")
	ErrMonthBoundaryViolation = errors.New("all transactions must be in the same month")
	ErrAlreadyGrouped         = errors.New("one or more transactions already belong to a group")
	ErrTransactionNotInGroup  = errors.New("one or more transactions do not belong to this group")
)

type TransactionGroup struct {
	ID             int32           `json:"id"`
	WorkspaceID    int32           `json:"workspaceId"`
	Name           string          `json:"name"`
	Month          string          `json:"month"`
	AutoDetected   bool            `json:"autoDetected"`
	LoanProviderID *int32          `json:"loanProviderId,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	// Derived fields (populated by repository queries)
	TotalAmount decimal.Decimal `json:"totalAmount"`
	ChildCount  int32           `json:"childCount"`
}

// AutoDetectionCandidate represents a consolidated_monthly provider with ungrouped transactions in a month
type AutoDetectionCandidate struct {
	ProviderID   int32
	ProviderName string
	Count        int32
}

// GroupOperationResult represents the result of a group delete/ungroup operation
type GroupOperationResult struct {
	GroupID          int32  `json:"groupId"`
	Mode             string `json:"mode"`
	ChildrenAffected int32  `json:"childrenAffected"`
}

func (g *TransactionGroup) Validate() error {
	if g.Name == "" {
		return ErrGroupNameEmpty
	}
	if !monthFormatRegex.MatchString(g.Month) {
		return ErrInvalidMonthFormat
	}
	return nil
}

type TransactionGroupRepository interface {
	Create(group *TransactionGroup) (*TransactionGroup, error)
	CreateWithAssignment(group *TransactionGroup, transactionIDs []int32) (*TransactionGroup, error)
	GetByID(workspaceID int32, id int32) (*TransactionGroup, error)
	GetGroupsByMonth(workspaceID int32, month string) ([]*TransactionGroup, error)
	UpdateName(workspaceID int32, id int32, name string) (*TransactionGroup, error)
	Delete(workspaceID int32, id int32) error
	AssignGroupToTransactions(workspaceID int32, groupID int32, transactionIDs []int32) error
	UnassignGroupFromTransactions(workspaceID int32, transactionIDs []int32) error
	UnassignAllFromGroup(workspaceID int32, groupID int32) (int64, error)
	DeleteGroupAndChildren(workspaceID int32, groupID int32) (int32, error)
	CountGroupChildren(workspaceID int32, groupID int32) (int32, error)
	GetUngroupedTransactionsByMonth(workspaceID int32, startDate, endDate time.Time) ([]*Transaction, error)
	GetConsolidatedProvidersByMonth(workspaceID int32, month string) ([]AutoDetectionCandidate, error)
	GetUngroupedTransactionIDsByProviderMonth(workspaceID int32, providerID int32, month string) ([]int32, error)
	GetAutoDetectedGroupByProviderMonth(workspaceID int32, providerID int32, month string) (*TransactionGroup, error)
}
