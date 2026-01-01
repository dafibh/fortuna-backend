package service

import (
	"strings"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
)

func TestCreateCategory_Success(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	name := "Groceries"

	category, err := categoryService.CreateCategory(workspaceID, name)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if category.Name != "Groceries" {
		t.Errorf("Expected name 'Groceries', got %s", category.Name)
	}

	if category.WorkspaceID != workspaceID {
		t.Errorf("Expected workspace ID %d, got %d", workspaceID, category.WorkspaceID)
	}
}

func TestCreateCategory_EmptyName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	_, err := categoryService.CreateCategory(workspaceID, "")
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}

	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateCategory_WhitespaceOnlyName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	_, err := categoryService.CreateCategory(workspaceID, "   ")
	if err == nil {
		t.Fatal("Expected error for whitespace-only name, got nil")
	}

	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestCreateCategory_TrimsName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	category, err := categoryService.CreateCategory(workspaceID, "  Groceries  ")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if category.Name != "Groceries" {
		t.Errorf("Expected trimmed name 'Groceries', got '%s'", category.Name)
	}
}

func TestCreateCategory_NameTooLong(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	// Create a name longer than MaxBudgetCategoryNameLength (100)
	longName := strings.Repeat("a", 101)

	_, err := categoryService.CreateCategory(workspaceID, longName)
	if err != domain.ErrNameTooLong {
		t.Errorf("Expected ErrNameTooLong, got %v", err)
	}
}

func TestCreateCategory_DuplicateName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	// Create first category
	_, err := categoryService.CreateCategory(workspaceID, "Groceries")
	if err != nil {
		t.Fatalf("Expected no error for first create, got %v", err)
	}

	// Try to create duplicate
	_, err = categoryService.CreateCategory(workspaceID, "Groceries")
	if err != domain.ErrBudgetCategoryAlreadyExists {
		t.Errorf("Expected ErrBudgetCategoryAlreadyExists, got %v", err)
	}
}

func TestGetCategories_Success(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	// Add some categories
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Groceries",
	})
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Transport",
	})

	categories, err := categoryService.GetCategories(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}
}

func TestGetCategories_EmptyList(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	categories, err := categoryService.GetCategories(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(categories))
	}
}

func TestGetCategories_ExcludesSoftDeleted(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	// Create and delete a category
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Groceries",
	})
	_ = categoryService.DeleteCategory(workspaceID, 1)

	// Create an active category
	_, _ = categoryService.CreateCategory(workspaceID, "Transport")

	categories, err := categoryService.GetCategories(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(categories) != 1 {
		t.Errorf("Expected 1 active category, got %d", len(categories))
	}

	if categories[0].Name != "Transport" {
		t.Errorf("Expected 'Transport', got %s", categories[0].Name)
	}
}

func TestGetCategoryByID_Success(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryID := int32(1)

	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          categoryID,
		WorkspaceID: workspaceID,
		Name:        "Groceries",
	})

	category, err := categoryService.GetCategoryByID(workspaceID, categoryID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if category.Name != "Groceries" {
		t.Errorf("Expected name 'Groceries', got %s", category.Name)
	}
}

func TestGetCategoryByID_NotFound(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	_, err := categoryService.GetCategoryByID(workspaceID, 999)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

func TestGetCategoryByID_WrongWorkspace(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	// Category belongs to workspace 1
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
	})

	// Try to get it from workspace 2
	_, err := categoryService.GetCategoryByID(2, 1)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound for wrong workspace, got %v", err)
	}
}

func TestUpdateCategory_Success(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	category, err := categoryService.UpdateCategory(workspaceID, 1, "New Name")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if category.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", category.Name)
	}
}

func TestUpdateCategory_TrimsName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	category, err := categoryService.UpdateCategory(workspaceID, 1, "  New Name  ")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if category.Name != "New Name" {
		t.Errorf("Expected trimmed name 'New Name', got '%s'", category.Name)
	}
}

func TestUpdateCategory_EmptyName(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	_, err := categoryService.UpdateCategory(workspaceID, 1, "")
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	_, err := categoryService.UpdateCategory(workspaceID, 999, "New Name")
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Groceries",
	})

	err := categoryService.DeleteCategory(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify category is soft-deleted (not found when querying active categories)
	_, err = categoryService.GetCategoryByID(workspaceID, 1)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound after soft delete, got %v", err)
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	err := categoryService.DeleteCategory(workspaceID, 999)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}

func TestCanDelete_NoTransactions(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)
	categoryRepo.AddBudgetCategory(&domain.BudgetCategory{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Groceries",
	})

	response, err := categoryService.CanDelete(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response.HasTransactions {
		t.Error("Expected HasTransactions to be false")
	}

	if response.TransactionCount != 0 {
		t.Errorf("Expected TransactionCount to be 0, got %d", response.TransactionCount)
	}
}

func TestCanDelete_NotFound(t *testing.T) {
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	categoryService := NewBudgetCategoryService(categoryRepo)

	workspaceID := int32(1)

	_, err := categoryService.CanDelete(workspaceID, 999)
	if err != domain.ErrBudgetCategoryNotFound {
		t.Errorf("Expected ErrBudgetCategoryNotFound, got %v", err)
	}
}
