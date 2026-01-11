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
	if len(name) > domain.MaxAccountNameLength {
		return nil, domain.ErrNameTooLong
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
func (s *AccountService) GetAccounts(workspaceID int32, includeArchived bool) ([]*domain.Account, error) {
	return s.accountRepo.GetAllByWorkspace(workspaceID, includeArchived)
}

// GetAccountByID retrieves an account by ID within a workspace
func (s *AccountService) GetAccountByID(workspaceID int32, id int32) (*domain.Account, error) {
	return s.accountRepo.GetByID(workspaceID, id)
}

// UpdateAccount updates an account's name (only name is editable)
func (s *AccountService) UpdateAccount(workspaceID int32, id int32, name string) (*domain.Account, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(name) > domain.MaxAccountNameLength {
		return nil, domain.ErrNameTooLong
	}

	return s.accountRepo.Update(workspaceID, id, name)
}

// DeleteAccount soft-deletes an account (sets deleted_at timestamp)
func (s *AccountService) DeleteAccount(workspaceID int32, id int32) error {
	// SoftDelete atomically checks existence and deletes, returning ErrAccountNotFound if not found
	return s.accountRepo.SoftDelete(workspaceID, id)
}

// CCOutstandingResult holds the aggregated CC outstanding data
// including total outstanding balance and per-account breakdown
type CCOutstandingResult struct {
	TotalOutstanding decimal.Decimal
	CCAccountCount   int32
	PerAccount       []*domain.PerAccountOutstanding
}

// GetCCOutstanding returns the total outstanding balance across all credit card accounts
// and a per-account breakdown. Outstanding balance is the sum of unpaid expenses.
// Returns CCOutstandingResult with zero values if no CC accounts exist.
func (s *AccountService) GetCCOutstanding(workspaceID int32) (*CCOutstandingResult, error) {
	summary, err := s.accountRepo.GetCCOutstandingSummary(workspaceID)
	if err != nil {
		return nil, err
	}

	perAccount, err := s.accountRepo.GetPerAccountOutstanding(workspaceID)
	if err != nil {
		return nil, err
	}

	return &CCOutstandingResult{
		TotalOutstanding: summary.TotalOutstanding,
		CCAccountCount:   summary.CCAccountCount,
		PerAccount:       perAccount,
	}, nil
}
