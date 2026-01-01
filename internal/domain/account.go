package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountType string
type AccountTemplate string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
)

const (
	TemplateBank       AccountTemplate = "bank"
	TemplateCash       AccountTemplate = "cash"
	TemplateEwallet    AccountTemplate = "ewallet"
	TemplateCreditCard AccountTemplate = "credit_card"
)

// TemplateToType maps account templates to their types
var TemplateToType = map[AccountTemplate]AccountType{
	TemplateBank:       AccountTypeAsset,
	TemplateCash:       AccountTypeAsset,
	TemplateEwallet:    AccountTypeAsset,
	TemplateCreditCard: AccountTypeLiability,
}

type Account struct {
	ID             int32           `json:"id"`
	WorkspaceID    int32           `json:"workspaceId"`
	Name           string          `json:"name"`
	AccountType    AccountType     `json:"accountType"`
	Template       AccountTemplate `json:"template"`
	InitialBalance decimal.Decimal `json:"initialBalance"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	DeletedAt      *time.Time      `json:"deletedAt,omitempty"`
}

// CCOutstandingSummary holds total CC outstanding across all accounts
type CCOutstandingSummary struct {
	TotalOutstanding decimal.Decimal `json:"totalOutstanding"`
	CCAccountCount   int32           `json:"ccAccountCount"`
}

// PerAccountOutstanding holds outstanding balance for a single CC account
type PerAccountOutstanding struct {
	AccountID          int32           `json:"accountId"`
	AccountName        string          `json:"accountName"`
	OutstandingBalance decimal.Decimal `json:"outstandingBalance"`
}

type AccountRepository interface {
	Create(account *Account) (*Account, error)
	GetByID(workspaceID int32, id int32) (*Account, error)
	GetAllByWorkspace(workspaceID int32, includeArchived bool) ([]*Account, error)
	Update(workspaceID int32, id int32, name string) (*Account, error)
	SoftDelete(workspaceID int32, id int32) error
	HardDelete(workspaceID int32, id int32) error
	GetCCOutstandingSummary(workspaceID int32) (*CCOutstandingSummary, error)
	GetPerAccountOutstanding(workspaceID int32) ([]*PerAccountOutstanding, error)
}
