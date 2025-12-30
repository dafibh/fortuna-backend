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
}

type AccountRepository interface {
	Create(account *Account) (*Account, error)
	GetByID(workspaceID int32, id int32) (*Account, error)
	GetAllByWorkspace(workspaceID int32) ([]*Account, error)
}
