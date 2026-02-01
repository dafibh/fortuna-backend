package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestTransactionGroupService_CreateGroup_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Create transactions in the same month
	tx1 := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	tx2 := &domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(30.00),
		TransactionDate: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx1)
	transactionRepo.AddTransaction(tx2)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	group, err := svc.CreateGroup(1, "Groceries", []int32{1, 2})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if group == nil {
		t.Fatal("expected group, got nil")
	}
	if group.Name != "Groceries" {
		t.Errorf("expected name 'Groceries', got %q", group.Name)
	}
	if group.Month != "2026-01" {
		t.Errorf("expected month '2026-01', got %q", group.Month)
	}
	if group.AutoDetected != false {
		t.Error("expected autoDetected false")
	}
	if group.ChildCount != 2 {
		t.Errorf("expected childCount 2, got %d", group.ChildCount)
	}
}

func TestTransactionGroupService_CreateGroup_EmptyName(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.CreateGroup(1, "", []int32{1})
	if err != domain.ErrGroupNameEmpty {
		t.Errorf("expected ErrGroupNameEmpty, got %v", err)
	}
}

func TestTransactionGroupService_CreateGroup_WhitespaceOnlyName(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.CreateGroup(1, "   ", []int32{1})
	if err != domain.ErrGroupNameEmpty {
		t.Errorf("expected ErrGroupNameEmpty, got %v", err)
	}
}

func TestTransactionGroupService_CreateGroup_TransactionNotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// No transactions added — IDs won't be found
	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.CreateGroup(1, "Test Group", []int32{999})
	if err != domain.ErrTransactionNotFound {
		t.Errorf("expected ErrTransactionNotFound, got %v", err)
	}
}

func TestTransactionGroupService_CreateGroup_MonthBoundaryViolation(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Create transactions in different months
	tx1 := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	tx2 := &domain.Transaction{
		ID:              2,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(30.00),
		TransactionDate: time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx1)
	transactionRepo.AddTransaction(tx2)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.CreateGroup(1, "Mixed Months", []int32{1, 2})
	if err != domain.ErrMonthBoundaryViolation {
		t.Errorf("expected ErrMonthBoundaryViolation, got %v", err)
	}
}

func TestTransactionGroupService_CreateGroup_AlreadyGrouped(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Create a transaction that is already in a group
	existingGroupID := int32(5)
	tx1 := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &existingGroupID,
	}
	transactionRepo.AddTransaction(tx1)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.CreateGroup(1, "Already Grouped", []int32{1})
	if err != domain.ErrAlreadyGrouped {
		t.Errorf("expected ErrAlreadyGrouped, got %v", err)
	}
}

func TestTransactionGroupService_CreateGroup_TrimsName(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	group, err := svc.CreateGroup(1, "  My Group  ", []int32{1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if group.Name != "My Group" {
		t.Errorf("expected trimmed name 'My Group', got %q", group.Name)
	}
}

func TestTransactionGroupService_RenameGroup_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Add existing group
	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Old Name",
		Month:       "2026-01",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	group, err := svc.RenameGroup(1, 1, "New Name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if group.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", group.Name)
	}
}

func TestTransactionGroupService_RenameGroup_EmptyName(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.RenameGroup(1, 1, "")
	if err != domain.ErrGroupNameEmpty {
		t.Errorf("expected ErrGroupNameEmpty, got %v", err)
	}
}

func TestTransactionGroupService_RenameGroup_NotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.RenameGroup(1, 999, "New Name")
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

// ==================== AddTransactionsToGroup ====================

func TestTransactionGroupService_AddTransactionsToGroup_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Existing group in January with 2 children
	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
		Month:       "2026-01",
		ChildCount:  2,
		TotalAmount: decimal.NewFromFloat(80.00),
	}
	groupRepo.AddGroup(group)

	// Ungrouped transaction in same month
	tx := &domain.Transaction{
		ID:              3,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(25.00),
		TransactionDate: time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	// Update mock state on assign
	groupRepo.AssignGroupToTransactionsFn = func(wsID int32, gID int32, txIDs []int32) error {
		g := groupRepo.Groups[gID]
		g.ChildCount += int32(len(txIDs))
		g.TotalAmount = g.TotalAmount.Add(decimal.NewFromFloat(25.00))
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	result, err := svc.AddTransactionsToGroup(1, 1, []int32{3})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ChildCount != 3 {
		t.Errorf("expected childCount 3, got %d", result.ChildCount)
	}
}

func TestTransactionGroupService_AddTransactionsToGroup_GroupNotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.AddTransactionsToGroup(1, 999, []int32{1})
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestTransactionGroupService_AddTransactionsToGroup_TransactionNotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.AddTransactionsToGroup(1, 1, []int32{999})
	if err != domain.ErrTransactionNotFound {
		t.Errorf("expected ErrTransactionNotFound, got %v", err)
	}
}

func TestTransactionGroupService_AddTransactionsToGroup_MonthBoundary(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Jan Group",
		Month:       "2026-01",
	})

	// Transaction in February
	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.AddTransactionsToGroup(1, 1, []int32{1})
	if err != domain.ErrMonthBoundaryViolation {
		t.Errorf("expected ErrMonthBoundaryViolation, got %v", err)
	}
}

func TestTransactionGroupService_AddTransactionsToGroup_AlreadyGrouped(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
	})

	existingGroupID := int32(2)
	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &existingGroupID,
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.AddTransactionsToGroup(1, 1, []int32{1})
	if err != domain.ErrAlreadyGrouped {
		t.Errorf("expected ErrAlreadyGrouped, got %v", err)
	}
}

// ==================== RemoveTransactionsFromGroup ====================

func TestTransactionGroupService_RemoveTransactionsFromGroup_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
		Month:       "2026-01",
		ChildCount:  3,
		TotalAmount: decimal.NewFromFloat(120.00),
	}
	groupRepo.AddGroup(group)

	groupID := int32(1)
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(40.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &groupID,
	}
	transactionRepo.AddTransaction(tx)

	// Update mock state on unassign
	groupRepo.UnassignGroupFromTransactionsFn = func(wsID int32, txIDs []int32) error {
		g := groupRepo.Groups[int32(1)]
		g.ChildCount -= int32(len(txIDs))
		g.TotalAmount = g.TotalAmount.Sub(decimal.NewFromFloat(40.00))
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	result, wasDeleted, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wasDeleted {
		t.Error("expected group NOT to be deleted")
	}
	if result.ChildCount != 2 {
		t.Errorf("expected childCount 2, got %d", result.ChildCount)
	}
}

func TestTransactionGroupService_RemoveTransactionsFromGroup_AutoDeleteEmpty(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Solo",
		Month:       "2026-01",
		ChildCount:  1,
		TotalAmount: decimal.NewFromFloat(50.00),
	}
	groupRepo.AddGroup(group)

	groupID := int32(1)
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &groupID,
	}
	transactionRepo.AddTransaction(tx)

	// Update mock state: removing last child
	groupRepo.UnassignGroupFromTransactionsFn = func(wsID int32, txIDs []int32) error {
		g := groupRepo.Groups[int32(1)]
		g.ChildCount = 0
		g.TotalAmount = decimal.Zero
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	result, wasDeleted, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !wasDeleted {
		t.Error("expected group to be auto-deleted")
	}
	if result != nil {
		t.Error("expected nil group when deleted")
	}
	// Verify group was deleted from mock
	if _, ok := groupRepo.Groups[1]; ok {
		t.Error("expected group to be removed from repository")
	}
}

func TestTransactionGroupService_RemoveTransactionsFromGroup_GroupNotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, _, err := svc.RemoveTransactionsFromGroup(1, 999, []int32{1})
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestTransactionGroupService_RemoveTransactionsFromGroup_TransactionNotInGroup(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
		ChildCount:  1,
	})

	// Transaction not in group 1
	otherGroupID := int32(2)
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &otherGroupID,
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, _, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != domain.ErrTransactionNotInGroup {
		t.Errorf("expected ErrTransactionNotInGroup, got %v", err)
	}
}

func TestTransactionGroupService_RemoveTransactionsFromGroup_UngroupedTransaction(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
		ChildCount:  1,
	})

	// Transaction with no group
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, _, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != domain.ErrTransactionNotInGroup {
		t.Errorf("expected ErrTransactionNotInGroup, got %v", err)
	}
}

// ==================== GetGroupsByMonth ====================

func TestTransactionGroupService_GetGroupsByMonth_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Jan Group 1",
		Month:       "2026-01",
	})
	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Jan Group 2",
		Month:       "2026-01",
	})
	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          3,
		WorkspaceID: 1,
		Name:        "Feb Group",
		Month:       "2026-02",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	groups, err := svc.GetGroupsByMonth(1, "2026-01")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestTransactionGroupService_GetGroupsByMonth_Empty(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	groups, err := svc.GetGroupsByMonth(1, "2026-01")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if groups == nil {
		// nil is acceptable for empty result from mock
	} else if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

// ==================== UngroupGroup ====================

func TestTransactionGroupService_UngroupGroup_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
		Month:       "2026-01",
		ChildCount:  3,
		TotalAmount: decimal.NewFromFloat(150.00),
	}
	groupRepo.AddGroup(group)

	// Mock UnassignAllFromGroup to return child count
	groupRepo.UnassignAllFromGroupFn = func(wsID int32, gID int32) (int64, error) {
		return 3, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	result, err := svc.UngroupGroup(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.GroupID != 1 {
		t.Errorf("expected groupId 1, got %d", result.GroupID)
	}
	if result.Mode != "ungroup" {
		t.Errorf("expected mode 'ungroup', got %q", result.Mode)
	}
	if result.ChildrenAffected != 3 {
		t.Errorf("expected childrenAffected 3, got %d", result.ChildrenAffected)
	}
	// Verify group was deleted
	if _, ok := groupRepo.Groups[1]; ok {
		t.Error("expected group to be removed from repository")
	}
}

func TestTransactionGroupService_UngroupGroup_NotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.UngroupGroup(1, 999)
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestTransactionGroupService_UngroupGroup_WrongWorkspace(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 2,
		Name:        "Other WS Group",
		Month:       "2026-01",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.UngroupGroup(1, 1)
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

// ==================== DeleteGroupWithChildren ====================

func TestTransactionGroupService_DeleteGroupWithChildren_Success(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Delete Me",
		Month:       "2026-01",
		ChildCount:  5,
		TotalAmount: decimal.NewFromFloat(250.00),
	}
	groupRepo.AddGroup(group)

	// Mock DeleteGroupAndChildren
	groupRepo.DeleteGroupAndChildrenFn = func(wsID int32, gID int32) (int32, error) {
		delete(groupRepo.Groups, gID)
		return 5, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	result, err := svc.DeleteGroupWithChildren(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.GroupID != 1 {
		t.Errorf("expected groupId 1, got %d", result.GroupID)
	}
	if result.Mode != "delete_all" {
		t.Errorf("expected mode 'delete_all', got %q", result.Mode)
	}
	if result.ChildrenAffected != 5 {
		t.Errorf("expected childrenAffected 5, got %d", result.ChildrenAffected)
	}
}

func TestTransactionGroupService_DeleteGroupWithChildren_NotFound(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.DeleteGroupWithChildren(1, 999)
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestTransactionGroupService_DeleteGroupWithChildren_WrongWorkspace(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 2,
		Name:        "Other WS",
		Month:       "2026-01",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	_, err := svc.DeleteGroupWithChildren(1, 1)
	if err != domain.ErrGroupNotFound {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

// ==================== EnsureAutoGroups ====================

func TestTransactionGroupService_EnsureAutoGroups_CreatesNewGroup(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Mock: 1 consolidated provider with 3 ungrouped transactions
	groupRepo.GetConsolidatedProvidersByMonthFn = func(wsID int32, month string) ([]domain.AutoDetectionCandidate, error) {
		return []domain.AutoDetectionCandidate{
			{ProviderID: 10, ProviderName: "SPaylater", Count: 3},
		}, nil
	}
	// Mock: no existing auto-detected group (default mock returns ErrGroupNotFound)
	groupRepo.GetAutoDetectedGroupByProviderMonthFn = func(wsID int32, providerID int32, month string) (*domain.TransactionGroup, error) {
		return nil, domain.ErrGroupNotFound
	}
	// Mock: return transaction IDs
	groupRepo.GetUngroupedTransactionIDsByProviderMonthFn = func(wsID int32, providerID int32, month string) ([]int32, error) {
		return []int32{100, 101, 102}, nil
	}

	var createdGroup *domain.TransactionGroup
	var assignedTxIDs []int32
	groupRepo.CreateFn = func(group *domain.TransactionGroup) (*domain.TransactionGroup, error) {
		createdGroup = group
		group.ID = 50
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
		groupRepo.Groups[group.ID] = group
		return group, nil
	}
	groupRepo.AssignGroupToTransactionsFn = func(wsID int32, gID int32, txIDs []int32) error {
		assignedTxIDs = txIDs
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if createdGroup == nil {
		t.Fatal("expected group to be created")
	}
	if createdGroup.Name != "SPaylater - February 2026" {
		t.Errorf("expected name 'SPaylater - February 2026', got %q", createdGroup.Name)
	}
	if createdGroup.AutoDetected != true {
		t.Error("expected autoDetected true")
	}
	if createdGroup.LoanProviderID == nil || *createdGroup.LoanProviderID != 10 {
		t.Error("expected loanProviderID 10")
	}
	if createdGroup.Month != "2026-02" {
		t.Errorf("expected month '2026-02', got %q", createdGroup.Month)
	}
	if len(assignedTxIDs) != 3 {
		t.Errorf("expected 3 transactions assigned, got %d", len(assignedTxIDs))
	}
}

func TestTransactionGroupService_EnsureAutoGroups_IdempotencyAddsToExisting(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Mock: 1 provider with ungrouped txns
	groupRepo.GetConsolidatedProvidersByMonthFn = func(wsID int32, month string) ([]domain.AutoDetectionCandidate, error) {
		return []domain.AutoDetectionCandidate{
			{ProviderID: 10, ProviderName: "SPaylater", Count: 2},
		}, nil
	}
	// Mock: existing auto-detected group
	existingGroup := &domain.TransactionGroup{
		ID:           42,
		WorkspaceID:  1,
		Name:         "SPaylater - February 2026",
		Month:        "2026-02",
		AutoDetected: true,
	}
	groupRepo.GetAutoDetectedGroupByProviderMonthFn = func(wsID int32, providerID int32, month string) (*domain.TransactionGroup, error) {
		return existingGroup, nil
	}
	// Mock: new ungrouped transaction IDs
	groupRepo.GetUngroupedTransactionIDsByProviderMonthFn = func(wsID int32, providerID int32, month string) ([]int32, error) {
		return []int32{200, 201}, nil
	}

	var assignedGroupID int32
	var assignedTxIDs []int32
	groupRepo.AssignGroupToTransactionsFn = func(wsID int32, gID int32, txIDs []int32) error {
		assignedGroupID = gID
		assignedTxIDs = txIDs
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if assignedGroupID != 42 {
		t.Errorf("expected assignment to existing group 42, got %d", assignedGroupID)
	}
	if len(assignedTxIDs) != 2 {
		t.Errorf("expected 2 transactions assigned, got %d", len(assignedTxIDs))
	}
}

func TestTransactionGroupService_EnsureAutoGroups_NoCandidates(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Mock: no candidates (default mock returns nil)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTransactionGroupService_EnsureAutoGroups_ErrorNeverPropagates(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Mock: query fails
	groupRepo.GetConsolidatedProvidersByMonthFn = func(wsID int32, month string) ([]domain.AutoDetectionCandidate, error) {
		return nil, domain.ErrGroupNotFound // simulate DB error
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected nil error (errors must not propagate), got %v", err)
	}
}

func TestTransactionGroupService_EnsureAutoGroups_MultipleProviders(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Mock: 2 providers
	groupRepo.GetConsolidatedProvidersByMonthFn = func(wsID int32, month string) ([]domain.AutoDetectionCandidate, error) {
		return []domain.AutoDetectionCandidate{
			{ProviderID: 10, ProviderName: "SPaylater", Count: 2},
			{ProviderID: 20, ProviderName: "Atome", Count: 3},
		}, nil
	}
	groupRepo.GetAutoDetectedGroupByProviderMonthFn = func(wsID int32, providerID int32, month string) (*domain.TransactionGroup, error) {
		return nil, domain.ErrGroupNotFound
	}
	groupRepo.GetUngroupedTransactionIDsByProviderMonthFn = func(wsID int32, providerID int32, month string) ([]int32, error) {
		if providerID == 10 {
			return []int32{100, 101}, nil
		}
		return []int32{200, 201, 202}, nil
	}

	createdCount := 0
	groupRepo.CreateFn = func(group *domain.TransactionGroup) (*domain.TransactionGroup, error) {
		createdCount++
		group.ID = int32(50 + createdCount)
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
		groupRepo.Groups[group.ID] = group
		return group, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if createdCount != 2 {
		t.Errorf("expected 2 groups created, got %d", createdCount)
	}
}

// ==================== WebSocket Event Publishing Tests ====================

func TestTransactionGroupService_CreateGroup_PublishesCreatedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	tx1 := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx1)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, err := svc.CreateGroup(1, "Groceries", []int32{1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	evt := mockPublisher.Events[0]
	if evt.WorkspaceID != 1 {
		t.Errorf("expected workspaceID 1, got %d", evt.WorkspaceID)
	}
	if evt.Event.Type != "transaction_group.created" {
		t.Errorf("expected event type 'transaction_group.created', got %q", evt.Event.Type)
	}
}

func TestTransactionGroupService_RenameGroup_PublishesUpdatedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Old Name",
		Month:       "2026-01",
	})

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, err := svc.RenameGroup(1, 1, "New Name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	if mockPublisher.Events[0].Event.Type != "transaction_group.updated" {
		t.Errorf("expected event type 'transaction_group.updated', got %q", mockPublisher.Events[0].Event.Type)
	}
}

func TestTransactionGroupService_UngroupGroup_PublishesDeletedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
		ChildCount:  2,
	})

	groupRepo.UnassignAllFromGroupFn = func(wsID int32, gID int32) (int64, error) {
		return 2, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, err := svc.UngroupGroup(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	evt := mockPublisher.Events[0]
	if evt.Event.Type != "transaction_group.deleted" {
		t.Errorf("expected event type 'transaction_group.deleted', got %q", evt.Event.Type)
	}
	// Verify mode is "ungroup" in payload
	payload, ok := evt.Event.Payload.(GroupDeletedPayload)
	if !ok {
		t.Fatalf("expected GroupDeletedPayload, got %T", evt.Event.Payload)
	}
	if payload.Mode != "ungroup" {
		t.Errorf("expected mode 'ungroup', got %q", payload.Mode)
	}
}

func TestTransactionGroupService_DeleteGroupWithChildren_PublishesDeletedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Delete Me",
		Month:       "2026-01",
		ChildCount:  3,
	})

	groupRepo.DeleteGroupAndChildrenFn = func(wsID int32, gID int32) (int32, error) {
		delete(groupRepo.Groups, gID)
		return 3, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, err := svc.DeleteGroupWithChildren(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	evt := mockPublisher.Events[0]
	if evt.Event.Type != "transaction_group.deleted" {
		t.Errorf("expected event type 'transaction_group.deleted', got %q", evt.Event.Type)
	}
	payload, ok := evt.Event.Payload.(GroupDeletedPayload)
	if !ok {
		t.Fatalf("expected GroupDeletedPayload, got %T", evt.Event.Payload)
	}
	if payload.Mode != "delete_all" {
		t.Errorf("expected mode 'delete_all', got %q", payload.Mode)
	}
}

func TestTransactionGroupService_AddTransactions_PublishesChildrenChangedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
		ChildCount:  1,
		TotalAmount: decimal.NewFromFloat(50.00),
	}
	groupRepo.AddGroup(group)

	tx := &domain.Transaction{
		ID:              3,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(25.00),
		TransactionDate: time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	groupRepo.AssignGroupToTransactionsFn = func(wsID int32, gID int32, txIDs []int32) error {
		g := groupRepo.Groups[gID]
		g.ChildCount += int32(len(txIDs))
		g.TotalAmount = g.TotalAmount.Add(decimal.NewFromFloat(25.00))
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, err := svc.AddTransactionsToGroup(1, 1, []int32{3})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	if mockPublisher.Events[0].Event.Type != "transaction_group.children_changed" {
		t.Errorf("expected event type 'transaction_group.children_changed', got %q", mockPublisher.Events[0].Event.Type)
	}
}

func TestTransactionGroupService_RemoveTransactions_PublishesChildrenChangedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test",
		Month:       "2026-01",
		ChildCount:  3,
		TotalAmount: decimal.NewFromFloat(120.00),
	}
	groupRepo.AddGroup(group)

	groupID := int32(1)
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(40.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &groupID,
	}
	transactionRepo.AddTransaction(tx)

	groupRepo.UnassignGroupFromTransactionsFn = func(wsID int32, txIDs []int32) error {
		g := groupRepo.Groups[int32(1)]
		g.ChildCount -= int32(len(txIDs))
		g.TotalAmount = g.TotalAmount.Sub(decimal.NewFromFloat(40.00))
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, _, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	if mockPublisher.Events[0].Event.Type != "transaction_group.children_changed" {
		t.Errorf("expected event type 'transaction_group.children_changed', got %q", mockPublisher.Events[0].Event.Type)
	}
}

func TestTransactionGroupService_RemoveLastChild_PublishesDeletedAutoEmpty(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Solo",
		Month:       "2026-01",
		ChildCount:  1,
		TotalAmount: decimal.NewFromFloat(50.00),
	}
	groupRepo.AddGroup(group)

	groupID := int32(1)
	tx := &domain.Transaction{
		ID:              5,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &groupID,
	}
	transactionRepo.AddTransaction(tx)

	groupRepo.UnassignGroupFromTransactionsFn = func(wsID int32, txIDs []int32) error {
		g := groupRepo.Groups[int32(1)]
		g.ChildCount = 0
		g.TotalAmount = decimal.Zero
		return nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	_, _, err := svc.RemoveTransactionsFromGroup(1, 1, []int32{5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	evt := mockPublisher.Events[0]
	if evt.Event.Type != "transaction_group.deleted" {
		t.Errorf("expected event type 'transaction_group.deleted', got %q", evt.Event.Type)
	}
	payload, ok := evt.Event.Payload.(GroupDeletedPayload)
	if !ok {
		t.Fatalf("expected GroupDeletedPayload, got %T", evt.Event.Payload)
	}
	if payload.Mode != "auto_empty" {
		t.Errorf("expected mode 'auto_empty', got %q", payload.Mode)
	}
}

func TestTransactionGroupService_EnsureAutoGroups_PublishesCreatedEvent(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	mockPublisher := testutil.NewMockEventPublisher()

	groupRepo.GetConsolidatedProvidersByMonthFn = func(wsID int32, month string) ([]domain.AutoDetectionCandidate, error) {
		return []domain.AutoDetectionCandidate{
			{ProviderID: 10, ProviderName: "SPaylater", Count: 2},
		}, nil
	}
	groupRepo.GetAutoDetectedGroupByProviderMonthFn = func(wsID int32, providerID int32, month string) (*domain.TransactionGroup, error) {
		return nil, domain.ErrGroupNotFound
	}
	groupRepo.GetUngroupedTransactionIDsByProviderMonthFn = func(wsID int32, providerID int32, month string) ([]int32, error) {
		return []int32{100, 101}, nil
	}
	groupRepo.CreateFn = func(group *domain.TransactionGroup) (*domain.TransactionGroup, error) {
		group.ID = 50
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
		groupRepo.Groups[group.ID] = group
		return group, nil
	}

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	svc.SetEventPublisher(mockPublisher)

	err := svc.EnsureAutoGroups(1, "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockPublisher.Events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(mockPublisher.Events))
	}
	if mockPublisher.Events[0].Event.Type != "transaction_group.created" {
		t.Errorf("expected event type 'transaction_group.created', got %q", mockPublisher.Events[0].Event.Type)
	}
}

func TestTransactionGroupService_NoPublisher_DoesNotPanic(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)
	// Deliberately NOT setting event publisher

	_, err := svc.CreateGroup(1, "Test", []int32{1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Test passes if no panic occurred
}

func TestTransactionGroupService_CreateGroup_WrongWorkspace(t *testing.T) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()

	// Transaction belongs to workspace 2
	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     2,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	svc := NewTransactionGroupService(groupRepo, transactionRepo)

	// Try to create group in workspace 1
	_, err := svc.CreateGroup(1, "Wrong WS", []int32{1})
	if err != domain.ErrTransactionNotFound {
		t.Errorf("expected ErrTransactionNotFound, got %v", err)
	}
}
