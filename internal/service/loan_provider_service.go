package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// LoanProviderService handles loan provider business logic
type LoanProviderService struct {
	providerRepo domain.LoanProviderRepository
}

// NewLoanProviderService creates a new LoanProviderService
func NewLoanProviderService(providerRepo domain.LoanProviderRepository) *LoanProviderService {
	return &LoanProviderService{providerRepo: providerRepo}
}

// CreateProviderInput contains input for creating a loan provider
type CreateProviderInput struct {
	Name                string
	CutoffDay           int32
	DefaultInterestRate decimal.Decimal
}

// CreateProvider creates a new loan provider
func (s *LoanProviderService) CreateProvider(workspaceID int32, input CreateProviderInput) (*domain.LoanProvider, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrLoanProviderNameEmpty
	}
	if len(name) > 100 {
		return nil, domain.ErrLoanProviderNameTooLong
	}

	// Validate cutoff day
	if input.CutoffDay < 1 || input.CutoffDay > 31 {
		return nil, domain.ErrInvalidCutoffDay
	}

	// Validate interest rate (0-100%)
	if input.DefaultInterestRate.LessThan(decimal.Zero) {
		return nil, domain.ErrInvalidInterestRate
	}
	if input.DefaultInterestRate.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrInterestRateTooHigh
	}

	provider := &domain.LoanProvider{
		WorkspaceID:         workspaceID,
		Name:                name,
		CutoffDay:           input.CutoffDay,
		DefaultInterestRate: input.DefaultInterestRate,
	}

	return s.providerRepo.Create(provider)
}

// GetProviders retrieves all loan providers for a workspace
func (s *LoanProviderService) GetProviders(workspaceID int32) ([]*domain.LoanProvider, error) {
	return s.providerRepo.GetAllByWorkspace(workspaceID)
}

// GetProviderByID retrieves a loan provider by ID within a workspace
func (s *LoanProviderService) GetProviderByID(workspaceID int32, id int32) (*domain.LoanProvider, error) {
	return s.providerRepo.GetByID(workspaceID, id)
}

// UpdateProviderInput contains input for updating a loan provider
type UpdateProviderInput struct {
	Name                string
	CutoffDay           int32
	DefaultInterestRate decimal.Decimal
}

// UpdateProvider updates a loan provider
func (s *LoanProviderService) UpdateProvider(workspaceID int32, id int32, input UpdateProviderInput) (*domain.LoanProvider, error) {
	// Verify provider exists
	existing, err := s.providerRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrLoanProviderNameEmpty
	}
	if len(name) > 100 {
		return nil, domain.ErrLoanProviderNameTooLong
	}

	// Validate cutoff day
	if input.CutoffDay < 1 || input.CutoffDay > 31 {
		return nil, domain.ErrInvalidCutoffDay
	}

	// Validate interest rate (0-100%)
	if input.DefaultInterestRate.LessThan(decimal.Zero) {
		return nil, domain.ErrInvalidInterestRate
	}
	if input.DefaultInterestRate.GreaterThan(decimal.NewFromInt(100)) {
		return nil, domain.ErrInterestRateTooHigh
	}

	existing.Name = name
	existing.CutoffDay = input.CutoffDay
	existing.DefaultInterestRate = input.DefaultInterestRate

	return s.providerRepo.Update(existing)
}

// DeleteProvider soft-deletes a loan provider
func (s *LoanProviderService) DeleteProvider(workspaceID int32, id int32) error {
	// Verify provider exists before deleting
	_, err := s.providerRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// NOTE: HasActiveLoans check will be added in Story 7-2 when loans table exists
	// For now, allow deletion without checking (no loans exist yet)

	return s.providerRepo.SoftDelete(workspaceID, id)
}
