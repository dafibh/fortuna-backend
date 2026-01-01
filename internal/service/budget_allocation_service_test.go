package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestGetAllocationsForMonth(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Setup test data
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
		{CategoryID: 3, CategoryName: "Entertainment", Allocated: decimal.Zero},
	})

	result, err := service.GetAllocationsForMonth(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Year != year {
		t.Errorf("expected year %d, got %d", year, result.Year)
	}
	if result.Month != month {
		t.Errorf("expected month %d, got %d", month, result.Month)
	}
	if len(result.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(result.Categories))
	}

	expectedTotal := decimal.NewFromInt(2500)
	if !result.TotalAllocated.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal.String(), result.TotalAllocated.String())
	}
}

func TestGetAllocationsForMonth_Empty(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	result, err := service.GetAllocationsForMonth(1, 2026, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(result.Categories))
	}
	if !result.TotalAllocated.Equal(decimal.Zero) {
		t.Errorf("expected total 0, got %s", result.TotalAllocated.String())
	}
}

func TestSetAllocation(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	amount := decimal.NewFromInt(2000)
	result, err := service.SetAllocation(workspaceID, categoryID, 2026, 1, amount)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.CategoryID != categoryID {
		t.Errorf("expected category ID %d, got %d", categoryID, result.CategoryID)
	}
	if result.CategoryName != "Food & Dining" {
		t.Errorf("expected category name 'Food & Dining', got '%s'", result.CategoryName)
	}
	if !result.Allocated.Equal(amount) {
		t.Errorf("expected allocated %s, got %s", amount.String(), result.Allocated.String())
	}
}

func TestSetAllocation_ZeroAmount(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Zero amount should be allowed
	amount := decimal.Zero
	result, err := service.SetAllocation(workspaceID, categoryID, 2026, 1, amount)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Allocated.Equal(decimal.Zero) {
		t.Errorf("expected allocated 0, got %s", result.Allocated.String())
	}
}

func TestSetAllocation_NegativeAmount(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Negative amount should fail
	amount := decimal.NewFromInt(-100)
	_, err := service.SetAllocation(workspaceID, categoryID, 2026, 1, amount)
	if err != domain.ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got: %v", err)
	}
}

func TestSetAllocation_CategoryNotFound(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	// Try to set allocation for non-existent category
	_, err := service.SetAllocation(1, 999, 2026, 1, decimal.NewFromInt(1000))
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("expected ErrBudgetCategoryNotFound, got: %v", err)
	}
}

func TestSetAllocation_CategoryFromDifferentWorkspace(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	// Add category to workspace 2
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: 2,
		Name:        "Food & Dining",
	})

	// Try to set allocation from workspace 1
	_, err := service.SetAllocation(1, 1, 2026, 1, decimal.NewFromInt(1000))
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("expected ErrBudgetCategoryNotFound, got: %v", err)
	}
}

func TestSetAllocations_Batch(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)

	// Add categories to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Transport",
	})

	// Setup the categories with allocations for the response
	allocationRepo.SetCategoriesWithAllocations(workspaceID, 2026, 1, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
	})

	allocations := []AllocationInput{
		{CategoryID: 1, Amount: decimal.NewFromInt(2000)},
		{CategoryID: 2, Amount: decimal.NewFromInt(500)},
	}

	result, err := service.SetAllocations(workspaceID, 2026, 1, allocations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expectedTotal := decimal.NewFromInt(2500)
	if !result.TotalAllocated.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal.String(), result.TotalAllocated.String())
	}
}

func TestSetAllocations_BatchWithInvalidCategory(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)

	// Only add one category
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	allocations := []AllocationInput{
		{CategoryID: 1, Amount: decimal.NewFromInt(2000)},
		{CategoryID: 999, Amount: decimal.NewFromInt(500)}, // Non-existent
	}

	_, err := service.SetAllocations(workspaceID, 2026, 1, allocations)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("expected ErrBudgetCategoryNotFound, got: %v", err)
	}
}

func TestSetAllocations_BatchWithNegativeAmount(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Transport",
	})

	allocations := []AllocationInput{
		{CategoryID: 1, Amount: decimal.NewFromInt(2000)},
		{CategoryID: 2, Amount: decimal.NewFromInt(-500)}, // Negative
	}

	_, err := service.SetAllocations(workspaceID, 2026, 1, allocations)
	if err != domain.ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got: %v", err)
	}
}

func TestDeleteAllocation(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	// Add category
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Add allocation
	allocationRepo.AddAllocation(&domain.BudgetAllocation{
		ID:          1,
		WorkspaceID: workspaceID,
		CategoryID:  categoryID,
		Year:        2026,
		Month:       1,
		Amount:      decimal.NewFromInt(2000),
	})

	err := service.DeleteAllocation(workspaceID, categoryID, 2026, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDeleteAllocation_CategoryNotFound(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	err := service.DeleteAllocation(1, 999, 2026, 1)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("expected ErrBudgetCategoryNotFound, got: %v", err)
	}
}

func TestGetMonthlyProgress(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Setup allocations
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
		{CategoryID: 3, CategoryName: "Entertainment", Allocated: decimal.NewFromInt(1000)},
	})

	// Setup spending
	allocationRepo.SetSpendingByCategory(workspaceID, year, month, []*domain.CategorySpending{
		{CategoryID: 1, Spent: decimal.NewFromInt(1850)}, // 92.5% - warning
		{CategoryID: 2, Spent: decimal.NewFromInt(650)},  // 130% - over
		{CategoryID: 3, Spent: decimal.NewFromInt(400)},  // 40% - healthy
	})

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Year != year {
		t.Errorf("expected year %d, got %d", year, result.Year)
	}
	if result.Month != month {
		t.Errorf("expected month %d, got %d", month, result.Month)
	}
	if len(result.Categories) != 3 {
		t.Errorf("expected 3 categories, got %d", len(result.Categories))
	}

	// Verify totals
	expectedTotalAllocated := decimal.NewFromInt(3500)
	if !result.TotalAllocated.Equal(expectedTotalAllocated) {
		t.Errorf("expected total allocated %s, got %s", expectedTotalAllocated.String(), result.TotalAllocated.String())
	}

	expectedTotalSpent := decimal.NewFromInt(2900)
	if !result.TotalSpent.Equal(expectedTotalSpent) {
		t.Errorf("expected total spent %s, got %s", expectedTotalSpent.String(), result.TotalSpent.String())
	}

	expectedTotalRemaining := decimal.NewFromInt(600)
	if !result.TotalRemaining.Equal(expectedTotalRemaining) {
		t.Errorf("expected total remaining %s, got %s", expectedTotalRemaining.String(), result.TotalRemaining.String())
	}
}

func TestGetMonthlyProgress_StatusThresholds(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Setup allocations - all 1000 for easy percentage calculation
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Healthy", Allocated: decimal.NewFromInt(1000)},
		{CategoryID: 2, CategoryName: "Warning", Allocated: decimal.NewFromInt(1000)},
		{CategoryID: 3, CategoryName: "Over", Allocated: decimal.NewFromInt(1000)},
	})

	// Setup spending
	allocationRepo.SetSpendingByCategory(workspaceID, year, month, []*domain.CategorySpending{
		{CategoryID: 1, Spent: decimal.NewFromInt(500)},  // 50% - healthy
		{CategoryID: 2, Spent: decimal.NewFromInt(850)},  // 85% - warning
		{CategoryID: 3, Spent: decimal.NewFromInt(1200)}, // 120% - over
	})

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Find categories by name and check status
	statusMap := make(map[string]domain.BudgetStatus)
	for _, cat := range result.Categories {
		statusMap[cat.CategoryName] = cat.Status
	}

	if statusMap["Healthy"] != domain.BudgetStatusHealthy {
		t.Errorf("expected Healthy to be healthy, got %s", statusMap["Healthy"])
	}
	if statusMap["Warning"] != domain.BudgetStatusWarning {
		t.Errorf("expected Warning to be warning, got %s", statusMap["Warning"])
	}
	if statusMap["Over"] != domain.BudgetStatusOver {
		t.Errorf("expected Over to be over, got %s", statusMap["Over"])
	}
}

func TestGetMonthlyProgress_ExactBoundaries(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Test exact boundary at 80%
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Exactly80", Allocated: decimal.NewFromInt(1000)},
		{CategoryID: 2, CategoryName: "Exactly100", Allocated: decimal.NewFromInt(1000)},
	})

	allocationRepo.SetSpendingByCategory(workspaceID, year, month, []*domain.CategorySpending{
		{CategoryID: 1, Spent: decimal.NewFromInt(800)},  // 80% - should be warning
		{CategoryID: 2, Spent: decimal.NewFromInt(1000)}, // 100% - should be over
	})

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	statusMap := make(map[string]domain.BudgetStatus)
	for _, cat := range result.Categories {
		statusMap[cat.CategoryName] = cat.Status
	}

	if statusMap["Exactly80"] != domain.BudgetStatusWarning {
		t.Errorf("expected 80%% to be warning, got %s", statusMap["Exactly80"])
	}
	if statusMap["Exactly100"] != domain.BudgetStatusOver {
		t.Errorf("expected 100%% to be over, got %s", statusMap["Exactly100"])
	}
}

func TestGetMonthlyProgress_ZeroAllocation(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Zero allocation should result in 0% percentage
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "ZeroBudget", Allocated: decimal.Zero},
	})

	allocationRepo.SetSpendingByCategory(workspaceID, year, month, []*domain.CategorySpending{
		{CategoryID: 1, Spent: decimal.NewFromInt(100)},
	})

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result.Categories))
	}

	cat := result.Categories[0]
	if !cat.Percentage.Equal(decimal.Zero) {
		t.Errorf("expected 0%% for zero allocation, got %s", cat.Percentage.String())
	}
	if cat.Status != domain.BudgetStatusHealthy {
		t.Errorf("expected healthy status for zero allocation, got %s", cat.Status)
	}
}

func TestGetMonthlyProgress_Empty(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	result, err := service.GetMonthlyProgress(1, 2026, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(result.Categories))
	}
	if !result.TotalAllocated.Equal(decimal.Zero) {
		t.Errorf("expected total allocated 0, got %s", result.TotalAllocated.String())
	}
	if !result.TotalSpent.Equal(decimal.Zero) {
		t.Errorf("expected total spent 0, got %s", result.TotalSpent.String())
	}
}

func TestGetMonthlyProgress_NoSpending(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// Allocation with no spending
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food", Allocated: decimal.NewFromInt(1000)},
	})

	// No spending set

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(result.Categories))
	}

	cat := result.Categories[0]
	if !cat.Spent.Equal(decimal.Zero) {
		t.Errorf("expected 0 spent, got %s", cat.Spent.String())
	}
	if !cat.Percentage.Equal(decimal.Zero) {
		t.Errorf("expected 0%%, got %s", cat.Percentage.String())
	}
	if !cat.Remaining.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("expected 1000 remaining, got %s", cat.Remaining.String())
	}
}

func TestGetCategoryTransactions(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)
	year := 2026
	month := 1

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// Setup mock transactions
	allocationRepo.GetCategoryTransactionsFn = func(wID int32, catID int32, y, m int) ([]*domain.CategoryTransaction, error) {
		if wID == workspaceID && catID == categoryID && y == year && m == month {
			return []*domain.CategoryTransaction{
				{ID: 1, Name: "Grocery Store", Amount: decimal.NewFromFloat(50.00), TransactionDate: "2026-01-15", AccountName: "DBS Debit"},
				{ID: 2, Name: "Restaurant", Amount: decimal.NewFromFloat(25.50), TransactionDate: "2026-01-10", AccountName: "DBS Debit"},
			}, nil
		}
		return []*domain.CategoryTransaction{}, nil
	}

	result, err := service.GetCategoryTransactions(workspaceID, categoryID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.CategoryID != categoryID {
		t.Errorf("expected category ID %d, got %d", categoryID, result.CategoryID)
	}
	if result.CategoryName != "Food & Dining" {
		t.Errorf("expected category name 'Food & Dining', got '%s'", result.CategoryName)
	}
	if len(result.Transactions) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(result.Transactions))
	}
}

func TestGetCategoryTransactions_CategoryNotFound(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	// Try to get transactions for non-existent category
	_, err := service.GetCategoryTransactions(1, 999, 2026, 1)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("expected ErrBudgetCategoryNotFound, got: %v", err)
	}
}

func TestGetCategoryTransactions_Empty(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	// Add category to mock
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Food & Dining",
	})

	// No transactions (default mock returns empty)
	result, err := service.GetCategoryTransactions(workspaceID, categoryID, 2026, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Transactions) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(result.Transactions))
	}
}

// Tests for month boundary scenarios (Story 4-5)


func TestGetMonthlyProgress_CopiesFromPreviousMonth(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 2 // February - new month with no allocations

	// Set count for target month to 0 (no allocations yet)
	allocationRepo.SetAllocationCount(workspaceID, year, month, 0)

	// Setup previous month (January) with allocations
	allocationRepo.SetAllocationCount(workspaceID, year, 1, 2)
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, 1, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
	})

	// Track if copy was called
	copyWasCalled := false
	allocationRepo.CopyAllocationsToMonthFn = func(wID int32, fromYear, fromMonth, toYear, toMonth int) error {
		copyWasCalled = true
		if wID != workspaceID || fromYear != year || fromMonth != 1 || toYear != year || toMonth != month {
			t.Errorf("CopyAllocationsToMonth called with wrong params")
		}
		// Simulate copy by updating the count
		allocationRepo.SetAllocationCount(wID, toYear, toMonth, 2)
		allocationRepo.SetCategoriesWithAllocations(wID, toYear, toMonth, []*domain.BudgetCategoryWithAllocation{
			{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
			{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
		})
		return nil
	}

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !copyWasCalled {
		t.Error("expected CopyAllocationsToMonth to be called")
	}

	if !result.CopiedFromPreviousMonth {
		t.Error("expected CopiedFromPreviousMonth to be true")
	}

	if !result.Initialized {
		t.Error("expected Initialized to be true")
	}
}

func TestGetMonthlyProgress_NoCopyWhenAllocationsExist(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 2

	// Set count for target month to non-zero (allocations already exist)
	allocationRepo.SetAllocationCount(workspaceID, year, month, 2)
	allocationRepo.SetCategoriesWithAllocations(workspaceID, year, month, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
	})

	copyWasCalled := false
	allocationRepo.CopyAllocationsToMonthFn = func(wID int32, fromYear, fromMonth, toYear, toMonth int) error {
		copyWasCalled = true
		return nil
	}

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if copyWasCalled {
		t.Error("expected CopyAllocationsToMonth NOT to be called when allocations exist")
	}

	if result.CopiedFromPreviousMonth {
		t.Error("expected CopiedFromPreviousMonth to be false when allocations already existed")
	}
}

func TestGetMonthlyProgress_FirstMonthEver(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)
	year := 2026
	month := 1

	// No allocations for target month
	allocationRepo.SetAllocationCount(workspaceID, year, month, 0)
	// No allocations for previous month (Dec 2025)
	allocationRepo.SetAllocationCount(workspaceID, 2025, 12, 0)

	allocationRepo.CopyAllocationsToMonthFn = func(wID int32, fromYear, fromMonth, toYear, toMonth int) error {
		// Copy is called but does nothing (no previous allocations)
		return nil
	}

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// No allocations copied because previous month was empty
	if result.CopiedFromPreviousMonth {
		t.Error("expected CopiedFromPreviousMonth to be false for first month ever")
	}

	if len(result.Categories) != 0 {
		t.Errorf("expected 0 categories for first month, got %d", len(result.Categories))
	}
}

func TestGetMonthlyProgress_YearBoundaryCopy(t *testing.T) {
	allocationRepo := testutil.NewMockBudgetAllocationRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	service := NewBudgetAllocationService(allocationRepo, categoryRepo)

	workspaceID := int32(1)

	// January 2026 - should copy from December 2025
	year := 2026
	month := 1

	// No allocations for Jan 2026
	allocationRepo.SetAllocationCount(workspaceID, year, month, 0)

	// Allocations exist for Dec 2025
	allocationRepo.SetAllocationCount(workspaceID, 2025, 12, 2)
	allocationRepo.SetCategoriesWithAllocations(workspaceID, 2025, 12, []*domain.BudgetCategoryWithAllocation{
		{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
		{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
	})

	copyFromYear := 0
	copyFromMonth := 0
	allocationRepo.CopyAllocationsToMonthFn = func(wID int32, fromYear, fromMonth, toYear, toMonth int) error {
		copyFromYear = fromYear
		copyFromMonth = fromMonth
		// Simulate successful copy
		allocationRepo.SetAllocationCount(wID, toYear, toMonth, 2)
		allocationRepo.SetCategoriesWithAllocations(wID, toYear, toMonth, []*domain.BudgetCategoryWithAllocation{
			{CategoryID: 1, CategoryName: "Food & Dining", Allocated: decimal.NewFromInt(2000)},
			{CategoryID: 2, CategoryName: "Transport", Allocated: decimal.NewFromInt(500)},
		})
		return nil
	}

	result, err := service.GetMonthlyProgress(workspaceID, year, month)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if copyFromYear != 2025 || copyFromMonth != 12 {
		t.Errorf("expected copy from Dec 2025, got %d/%d", copyFromYear, copyFromMonth)
	}

	if !result.CopiedFromPreviousMonth {
		t.Error("expected CopiedFromPreviousMonth to be true")
	}
}
