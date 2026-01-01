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
