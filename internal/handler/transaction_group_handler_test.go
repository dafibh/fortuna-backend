package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func setupGroupHandler() (*TransactionGroupHandler, *testutil.MockTransactionGroupRepository, *testutil.MockTransactionRepository) {
	groupRepo := testutil.NewMockTransactionGroupRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	groupService := service.NewTransactionGroupService(groupRepo, transactionRepo)
	handler := NewTransactionGroupHandler(groupService)
	return handler, groupRepo, transactionRepo
}

func createGroupContext(method, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	e := echo.New()
	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))
	return c, rec
}

func TestTransactionGroupHandler_CreateGroup_Success(t *testing.T) {
	handler, _, transactionRepo := setupGroupHandler()

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

	reqBody := CreateGroupRequest{
		Name:           "Groceries",
		TransactionIDs: []int32{1, 2},
	}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups", reqBody)

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.Name != "Groceries" {
		t.Errorf("expected name 'Groceries', got %q", response.Name)
	}
	if response.Month != "2026-01" {
		t.Errorf("expected month '2026-01', got %q", response.Month)
	}
	if response.ChildCount != 2 {
		t.Errorf("expected childCount 2, got %d", response.ChildCount)
	}
}

func TestTransactionGroupHandler_CreateGroup_MissingWorkspace(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := CreateGroupRequest{
		Name:           "Test",
		TransactionIDs: []int32{1},
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction-groups", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Not setting workspace ID

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestTransactionGroupHandler_CreateGroup_EmptyTransactionIDs(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := CreateGroupRequest{
		Name:           "Test",
		TransactionIDs: []int32{},
	}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups", reqBody)

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_CreateGroup_InvalidJSON(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction-groups", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ctx := context.WithValue(c.Request().Context(), middleware.WorkspaceIDKey, int32(1))
	c.SetRequest(c.Request().WithContext(ctx))

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_CreateGroup_MonthBoundaryViolation(t *testing.T) {
	handler, _, transactionRepo := setupGroupHandler()

	// Transactions in different months
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

	reqBody := CreateGroupRequest{
		Name:           "Mixed",
		TransactionIDs: []int32{1, 2},
	}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups", reqBody)

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}

	var problem ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if problem.Type != ErrorTypeMonthBoundary {
		t.Errorf("expected error type %q, got %q", ErrorTypeMonthBoundary, problem.Type)
	}
}

func TestTransactionGroupHandler_CreateGroup_AlreadyGrouped(t *testing.T) {
	handler, _, transactionRepo := setupGroupHandler()

	existingGroupID := int32(5)
	tx1 := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		GroupID:         &existingGroupID,
	}
	transactionRepo.AddTransaction(tx1)

	reqBody := CreateGroupRequest{
		Name:           "Duplicate",
		TransactionIDs: []int32{1},
	}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups", reqBody)

	err := handler.CreateGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}

	var problem ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if problem.Type != ErrorTypeAlreadyGrouped {
		t.Errorf("expected error type %q, got %q", ErrorTypeAlreadyGrouped, problem.Type)
	}
}

func TestTransactionGroupHandler_RenameGroup_Success(t *testing.T) {
	handler, groupRepo, _ := setupGroupHandler()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Old Name",
		Month:       "2026-01",
	})

	reqBody := RenameGroupRequest{Name: "New Name"}
	c, rec := createGroupContext(http.MethodPut, "/api/v1/transaction-groups/1", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.RenameGroup(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", response.Name)
	}
}

func TestTransactionGroupHandler_RenameGroup_NotFound(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := RenameGroupRequest{Name: "New Name"}
	c, rec := createGroupContext(http.MethodPut, "/api/v1/transaction-groups/999", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := handler.RenameGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestTransactionGroupHandler_RenameGroup_EmptyName(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := RenameGroupRequest{Name: ""}
	c, rec := createGroupContext(http.MethodPut, "/api/v1/transaction-groups/1", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.RenameGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ==================== AddTransactions ====================

func TestTransactionGroupHandler_AddTransactions_Success(t *testing.T) {
	handler, groupRepo, transactionRepo := setupGroupHandler()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
		Month:       "2026-01",
		ChildCount:  2,
		TotalAmount: decimal.NewFromFloat(80.00),
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
		return nil
	}

	reqBody := MembershipRequest{TransactionIDs: []int32{3}}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.AddTransactions(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.ChildCount != 3 {
		t.Errorf("expected childCount 3, got %d", response.ChildCount)
	}
}

func TestTransactionGroupHandler_AddTransactions_EmptyIDs(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := MembershipRequest{TransactionIDs: []int32{}}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.AddTransactions(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_AddTransactions_MonthBoundary(t *testing.T) {
	handler, groupRepo, transactionRepo := setupGroupHandler()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Jan Group",
		Month:       "2026-01",
	})

	tx := &domain.Transaction{
		ID:              1,
		WorkspaceID:     1,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionDate: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
	}
	transactionRepo.AddTransaction(tx)

	reqBody := MembershipRequest{TransactionIDs: []int32{1}}
	c, rec := createGroupContext(http.MethodPost, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.AddTransactions(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
}

// ==================== RemoveTransactions ====================

func TestTransactionGroupHandler_RemoveTransactions_Success(t *testing.T) {
	handler, groupRepo, transactionRepo := setupGroupHandler()

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

	groupRepo.UnassignGroupFromTransactionsFn = func(wsID int32, txIDs []int32) error {
		g := groupRepo.Groups[int32(1)]
		g.ChildCount -= int32(len(txIDs))
		return nil
	}

	reqBody := MembershipRequest{TransactionIDs: []int32{5}}
	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.RemoveTransactions(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.ChildCount != 2 {
		t.Errorf("expected childCount 2, got %d", response.ChildCount)
	}
}

func TestTransactionGroupHandler_RemoveTransactions_AutoDelete(t *testing.T) {
	handler, groupRepo, transactionRepo := setupGroupHandler()

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

	reqBody := MembershipRequest{TransactionIDs: []int32{5}}
	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.RemoveTransactions(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response GroupDeletedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !response.Deleted {
		t.Error("expected deleted=true")
	}
	if response.GroupID != 1 {
		t.Errorf("expected groupId 1, got %d", response.GroupID)
	}
}

func TestTransactionGroupHandler_RemoveTransactions_EmptyIDs(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := MembershipRequest{TransactionIDs: []int32{}}
	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1/transactions", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.RemoveTransactions(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// ==================== GetGroupsByMonth ====================

func TestTransactionGroupHandler_GetGroupsByMonth_Success(t *testing.T) {
	handler, groupRepo, _ := setupGroupHandler()

	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Group 1",
		Month:       "2026-01",
		ChildCount:  2,
	})
	groupRepo.AddGroup(&domain.TransactionGroup{
		ID:          2,
		WorkspaceID: 1,
		Name:        "Group 2",
		Month:       "2026-01",
		ChildCount:  3,
	})

	c, rec := createGroupContext(http.MethodGet, "/api/v1/transaction-groups?month=2026-01", nil)
	c.QueryParams().Set("month", "2026-01")

	err := handler.GetGroupsByMonth(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var responses []GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(responses) != 2 {
		t.Errorf("expected 2 groups, got %d", len(responses))
	}
}

func TestTransactionGroupHandler_GetGroupsByMonth_MissingMonth(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodGet, "/api/v1/transaction-groups", nil)

	err := handler.GetGroupsByMonth(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_GetGroupsByMonth_Empty(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodGet, "/api/v1/transaction-groups?month=2026-03", nil)
	c.QueryParams().Set("month", "2026-03")

	err := handler.GetGroupsByMonth(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var responses []GroupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(responses) != 0 {
		t.Errorf("expected 0 groups, got %d", len(responses))
	}
}

// ==================== DeleteGroup (Ungroup + DeleteAll) ====================

func TestTransactionGroupHandler_DeleteGroup_Ungroup_Success(t *testing.T) {
	handler, groupRepo, _ := setupGroupHandler()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Groceries",
		Month:       "2026-01",
		ChildCount:  3,
		TotalAmount: decimal.NewFromFloat(150.00),
	}
	groupRepo.AddGroup(group)

	groupRepo.UnassignAllFromGroupFn = func(wsID int32, gID int32) (int64, error) {
		return 3, nil
	}

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1?mode=ungroup", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.QueryParams().Set("mode", "ungroup")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response domain.GroupOperationResult
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.GroupID != 1 {
		t.Errorf("expected groupId 1, got %d", response.GroupID)
	}
	if response.Mode != "ungroup" {
		t.Errorf("expected mode 'ungroup', got %q", response.Mode)
	}
	if response.ChildrenAffected != 3 {
		t.Errorf("expected childrenAffected 3, got %d", response.ChildrenAffected)
	}
}

func TestTransactionGroupHandler_DeleteGroup_DeleteAll_Success(t *testing.T) {
	handler, groupRepo, _ := setupGroupHandler()

	group := &domain.TransactionGroup{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Delete Me",
		Month:       "2026-01",
		ChildCount:  5,
		TotalAmount: decimal.NewFromFloat(250.00),
	}
	groupRepo.AddGroup(group)

	groupRepo.DeleteGroupAndChildrenFn = func(wsID int32, gID int32) (int32, error) {
		delete(groupRepo.Groups, gID)
		return 5, nil
	}

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1?mode=delete_all", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.QueryParams().Set("mode", "delete_all")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response domain.GroupOperationResult
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if response.GroupID != 1 {
		t.Errorf("expected groupId 1, got %d", response.GroupID)
	}
	if response.Mode != "delete_all" {
		t.Errorf("expected mode 'delete_all', got %q", response.Mode)
	}
	if response.ChildrenAffected != 5 {
		t.Errorf("expected childrenAffected 5, got %d", response.ChildrenAffected)
	}
}

func TestTransactionGroupHandler_DeleteGroup_InvalidMode(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1?mode=invalid", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.QueryParams().Set("mode", "invalid")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_DeleteGroup_MissingMode(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/1", nil)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_DeleteGroup_InvalidID(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/abc?mode=ungroup", nil)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.QueryParams().Set("mode", "ungroup")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestTransactionGroupHandler_DeleteGroup_NotFound(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	c, rec := createGroupContext(http.MethodDelete, "/api/v1/transaction-groups/999?mode=ungroup", nil)
	c.SetParamNames("id")
	c.SetParamValues("999")
	c.QueryParams().Set("mode", "ungroup")

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestTransactionGroupHandler_DeleteGroup_MissingWorkspace(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	body, _ := json.Marshal(nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/transaction-groups/1?mode=ungroup", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	c.QueryParams().Set("mode", "ungroup")
	// Not setting workspace ID

	err := handler.DeleteGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestTransactionGroupHandler_RenameGroup_InvalidID(t *testing.T) {
	handler, _, _ := setupGroupHandler()

	reqBody := RenameGroupRequest{Name: "New Name"}
	c, rec := createGroupContext(http.MethodPut, "/api/v1/transaction-groups/abc", reqBody)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.RenameGroup(c)
	if err != nil {
		t.Fatalf("expected nil error (error in response), got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
