package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// AccountService handles account-related business logic
type AccountService struct {
	accountRepo domain.AccountRepository
}

// NewAccountService creates a new AccountService
func NewAccountService(accountRepo domain.AccountRepository) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

// CreateAccountInput holds the input for creating an account
type CreateAccountInput struct {
	Name           string
	Template       domain.AccountTemplate
	InitialBalance decimal.Decimal
}

// CreateAccount creates a new account with template-to-type mapping
func (s *AccountService) CreateAccount(workspaceID int32, input CreateAccountInput) (*domain.Account, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}

	// Determine account type from template
	accountType, ok := domain.TemplateToType[input.Template]
	if !ok {
		return nil, domain.ErrInvalidTemplate
	}

	account := &domain.Account{
		WorkspaceID:    workspaceID,
		Name:           name,
		AccountType:    accountType,
		Template:       input.Template,
		InitialBalance: input.InitialBalance,
	}

	return s.accountRepo.Create(account)
}

// GetAccounts retrieves all accounts for a workspace
func (s *AccountService) GetAccounts(workspaceID int32) ([]*domain.Account, error) {
	return s.accountRepo.GetAllByWorkspace(workspaceID)
}

// GetAccountByID retrieves an account by ID within a workspace
func (s *AccountService) GetAccountByID(workspaceID int32, id int32) (*domain.Account, error) {
	return s.accountRepo.GetByID(workspaceID, id)
}
