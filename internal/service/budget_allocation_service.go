package service

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/util"
	"github.com/shopspring/decimal"
)

// BudgetAllocationService handles budget allocation business logic
type BudgetAllocationService struct {
	allocationRepo domain.BudgetAllocationRepository
	categoryRepo   domain.BudgetCategoryRepository
}

// NewBudgetAllocationService creates a new BudgetAllocationService
func NewBudgetAllocationService(
	allocationRepo domain.BudgetAllocationRepository,
	categoryRepo domain.BudgetCategoryRepository,
) *BudgetAllocationService {
	return &BudgetAllocationService{
		allocationRepo: allocationRepo,
		categoryRepo:   categoryRepo,
	}
}

// AllocationInput represents a single allocation to set
type AllocationInput struct {
	CategoryID int32
	Amount     decimal.Decimal
}

// BudgetMonthResponse contains all budget allocation info for a month
type BudgetMonthResponse struct {
	Year           int                                   `json:"year"`
	Month          int                                   `json:"month"`
	TotalAllocated decimal.Decimal                       `json:"totalAllocated"`
	Categories     []*domain.BudgetCategoryWithAllocation `json:"categories"`
}

// GetAllocationsForMonth retrieves all categories with their allocations for a month
func (s *BudgetAllocationService) GetAllocationsForMonth(workspaceID int32, year, month int) (*BudgetMonthResponse, error) {
	categories, err := s.allocationRepo.GetCategoriesWithAllocations(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Calculate total allocated
	total := decimal.Zero
	for _, cat := range categories {
		total = total.Add(cat.Allocated)
	}

	return &BudgetMonthResponse{
		Year:           year,
		Month:          month,
		TotalAllocated: total,
		Categories:     categories,
	}, nil
}

// SetAllocation sets (or updates) a single budget allocation
func (s *BudgetAllocationService) SetAllocation(workspaceID int32, categoryID int32, year, month int, amount decimal.Decimal) (*domain.BudgetCategoryWithAllocation, error) {
	// Validate amount is non-negative
	if amount.IsNegative() {
		return nil, domain.ErrInvalidAmount
	}

	// Validate category exists and belongs to workspace
	category, err := s.categoryRepo.GetByID(workspaceID, categoryID)
	if err != nil {
		return nil, err
	}

	// Upsert the allocation
	allocation := &domain.BudgetAllocation{
		WorkspaceID: workspaceID,
		CategoryID:  categoryID,
		Year:        year,
		Month:       month,
		Amount:      amount,
	}

	_, err = s.allocationRepo.Upsert(allocation)
	if err != nil {
		return nil, err
	}

	return &domain.BudgetCategoryWithAllocation{
		CategoryID:   categoryID,
		CategoryName: category.Name,
		Allocated:    amount,
	}, nil
}

// SetAllocations sets multiple budget allocations in a batch (atomic transaction)
func (s *BudgetAllocationService) SetAllocations(workspaceID int32, year, month int, allocations []AllocationInput) (*BudgetMonthResponse, error) {
	// First pass: validate all inputs before making any changes
	domainAllocations := make([]*domain.BudgetAllocation, len(allocations))
	for i, input := range allocations {
		// Validate amount is non-negative
		if input.Amount.IsNegative() {
			return nil, domain.ErrInvalidAmount
		}

		// Validate category exists and belongs to workspace
		_, err := s.categoryRepo.GetByID(workspaceID, input.CategoryID)
		if err != nil {
			return nil, err
		}

		domainAllocations[i] = &domain.BudgetAllocation{
			WorkspaceID: workspaceID,
			CategoryID:  input.CategoryID,
			Year:        year,
			Month:       month,
			Amount:      input.Amount,
		}
	}

	// Second pass: upsert all allocations atomically
	if err := s.allocationRepo.UpsertBatch(domainAllocations); err != nil {
		return nil, err
	}

	// Return updated month response
	return s.GetAllocationsForMonth(workspaceID, year, month)
}

// DeleteAllocation removes a budget allocation
func (s *BudgetAllocationService) DeleteAllocation(workspaceID int32, categoryID int32, year, month int) error {
	// Validate category exists and belongs to workspace
	_, err := s.categoryRepo.GetByID(workspaceID, categoryID)
	if err != nil {
		return err
	}

	return s.allocationRepo.Delete(workspaceID, categoryID, year, month)
}

// GetCategoryTransactions retrieves transactions for a specific category in a month
func (s *BudgetAllocationService) GetCategoryTransactions(workspaceID int32, categoryID int32, year, month int) (*domain.CategoryTransactionsResponse, error) {
	// Validate category exists and belongs to workspace
	category, err := s.categoryRepo.GetByID(workspaceID, categoryID)
	if err != nil {
		return nil, err
	}

	transactions, err := s.allocationRepo.GetCategoryTransactions(workspaceID, categoryID, year, month)
	if err != nil {
		return nil, err
	}

	return &domain.CategoryTransactionsResponse{
		CategoryID:   categoryID,
		CategoryName: category.Name,
		Transactions: transactions,
	}, nil
}

// GetMonthlyProgress retrieves budget progress for all categories in a month
func (s *BudgetAllocationService) GetMonthlyProgress(workspaceID int32, year, month int) (*domain.MonthlyBudgetSummary, error) {
	// Check if this month has any allocations (lazy initialization check)
	existingCount, err := s.allocationRepo.CountAllocationsForMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	copiedFromPreviousMonth := false

	// If no allocations, try to copy from previous month
	if existingCount == 0 {
		prevYear, prevMonth := util.PreviousMonth(year, month)
		err := s.allocationRepo.CopyAllocationsToMonth(workspaceID, prevYear, prevMonth, year, month)
		if err != nil {
			// Log but don't fail - might be first month or no previous allocations
			// In production, we'd log this error
		} else {
			// Check if we actually copied anything
			newCount, _ := s.allocationRepo.CountAllocationsForMonth(workspaceID, year, month)
			copiedFromPreviousMonth = newCount > 0
		}
	}

	// Get all categories with allocations
	allocations, err := s.allocationRepo.GetCategoriesWithAllocations(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Get spending by category
	spending, err := s.allocationRepo.GetSpendingByCategory(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Build spending map for quick lookup
	spentMap := make(map[int32]decimal.Decimal)
	for _, sp := range spending {
		spentMap[sp.CategoryID] = sp.Spent
	}

	// Calculate progress for each category
	categories := make([]*domain.BudgetProgress, 0, len(allocations))
	totalAllocated := decimal.Zero
	totalSpent := decimal.Zero

	hundred := decimal.NewFromInt(100)
	eightyPercent := decimal.NewFromInt(80)

	for _, alloc := range allocations {
		spent := spentMap[alloc.CategoryID]
		if spent.IsZero() {
			spent = decimal.Zero
		}

		remaining := alloc.Allocated.Sub(spent)

		var percentage decimal.Decimal
		if alloc.Allocated.IsPositive() {
			percentage = spent.Div(alloc.Allocated).Mul(hundred)
		} else {
			percentage = decimal.Zero
		}

		status := domain.BudgetStatusHealthy
		if percentage.GreaterThanOrEqual(hundred) {
			status = domain.BudgetStatusOver
		} else if percentage.GreaterThanOrEqual(eightyPercent) {
			status = domain.BudgetStatusWarning
		}

		categories = append(categories, &domain.BudgetProgress{
			CategoryID:   alloc.CategoryID,
			CategoryName: alloc.CategoryName,
			Allocated:    alloc.Allocated,
			Spent:        spent,
			Remaining:    remaining,
			Percentage:   percentage.Round(2),
			Status:       status,
		})

		totalAllocated = totalAllocated.Add(alloc.Allocated)
		totalSpent = totalSpent.Add(spent)
	}

	return &domain.MonthlyBudgetSummary{
		Year:                    year,
		Month:                   month,
		TotalAllocated:          totalAllocated,
		TotalSpent:              totalSpent,
		TotalRemaining:          totalAllocated.Sub(totalSpent),
		Categories:              categories,
		Initialized:             true,
		CopiedFromPreviousMonth: copiedFromPreviousMonth,
		IsHistorical:            util.IsHistoricalMonth(year, month),
	}, nil
}
