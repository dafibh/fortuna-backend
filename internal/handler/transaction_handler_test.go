package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func TestCreateTransaction_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountID := int32(1)

	// Add account to mock
	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "Groceries", "amount": "150.00", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response TransactionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "Groceries" {
		t.Errorf("Expected name 'Groceries', got %s", response.Name)
	}

	if response.Amount != "150.00" {
		t.Errorf("Expected amount '150.00', got %s", response.Amount)
	}

	if response.Type != "expense" {
		t.Errorf("Expected type 'expense', got %s", response.Type)
	}

	if !response.IsPaid {
		t.Error("Expected is_paid to default to true")
	}
}

func TestCreateTransaction_WithDate(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "Past Transaction", "amount": "100.00", "type": "expense", "date": "2025-01-15"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response TransactionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TransactionDate != "2025-01-15" {
		t.Errorf("Expected date '2025-01-15', got %s", response.TransactionDate)
	}
}

func TestCreateTransaction_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	reqBody := `{"accountId": 1, "name": "Test", "amount": "100.00", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCreateTransaction_MissingName(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "", "amount": "100.00", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "name" {
		t.Error("Expected validation error for 'name' field")
	}
}

func TestCreateTransaction_InvalidAmount(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "Test", "amount": "not-a-number", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateTransaction_ZeroAmount(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "Test", "amount": "0", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "amount" {
		t.Error("Expected validation error for 'amount' field")
	}
}

func TestCreateTransaction_InvalidType(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Account",
	})

	reqBody := `{"accountId": 1, "name": "Test", "amount": "100.00", "type": "invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "type" {
		t.Error("Expected validation error for 'type' field")
	}
}

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	reqBody := `{"accountId": 999, "name": "Test", "amount": "100.00", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "accountId" {
		t.Error("Expected validation error for 'accountId' field")
	}
}

func TestCreateTransaction_ZeroAccountId(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	reqBody := `{"accountId": 0, "name": "Test", "amount": "100.00", "type": "expense"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransaction(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "accountId" {
		t.Error("Expected validation error for 'accountId' field")
	}
}

func TestGetTransactions_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	// Add test transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Transaction 1",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
	})
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Transaction 2",
		Amount:      decimal.NewFromFloat(200.00),
		Type:        domain.TransactionTypeIncome,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetTransactions(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response PaginatedTransactionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Data) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(response.Data))
	}

	if response.TotalItems != 2 {
		t.Errorf("Expected totalItems 2, got %d", response.TotalItems)
	}

	if response.Page != 1 {
		t.Errorf("Expected page 1, got %d", response.Page)
	}
}

func TestGetTransactions_EmptyList(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetTransactions(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response PaginatedTransactionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Data) != 0 {
		t.Errorf("Expected 0 transactions, got %d", len(response.Data))
	}

	if response.TotalItems != 0 {
		t.Errorf("Expected totalItems 0, got %d", response.TotalItems)
	}
}

func TestGetTransactions_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	// Add transaction to workspace 1
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   1,
		Name:        "Workspace 1 Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
	})

	// Add transaction to workspace 2
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          2,
		WorkspaceID: 2,
		AccountID:   2,
		Name:        "Workspace 2 Transaction",
		Amount:      decimal.NewFromFloat(200.00),
		Type:        domain.TransactionTypeExpense,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Request from workspace 1 - should only see workspace 1's transaction
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetTransactions(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response PaginatedTransactionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(response.Data))
	}

	if response.Data[0].Name != "Workspace 1 Transaction" {
		t.Errorf("Expected 'Workspace 1 Transaction', got %s", response.Data[0].Name)
	}
}

func TestTogglePaidStatus_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	// Add a paid transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/1/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response TransactionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.IsPaid {
		t.Error("Expected is_paid to be false after toggle")
	}
}

func TestTogglePaidStatus_UnpaidToPaid(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	// Add an unpaid transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: workspaceID,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      false,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/1/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response TransactionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.IsPaid {
		t.Error("Expected is_paid to be true after toggle")
	}
}

func TestTogglePaidStatus_InvalidID(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/invalid/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestTogglePaidStatus_NotFound(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/999/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestTogglePaidStatus_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/1/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestTogglePaidStatus_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	// Transaction belongs to workspace 1
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:          1,
		WorkspaceID: 1,
		AccountID:   1,
		Name:        "Test Transaction",
		Amount:      decimal.NewFromFloat(100.00),
		Type:        domain.TransactionTypeExpense,
		IsPaid:      true,
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/1/toggle-paid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	// Request from workspace 2 - should not be able to toggle workspace 1's transaction
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 2)

	err := handler.TogglePaidStatus(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for workspace isolation, got %d", rec.Code)
	}
}

// ============================================
// Transfer Handler Tests
// ============================================

func TestCreateTransfer_Success(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	// Add source and destination accounts
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	reqBody := `{"fromAccountId": 1, "toAccountId": 2, "amount": "500.00", "date": "2025-01-15", "notes": "Monthly savings"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response TransferResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Validate from transaction
	if response.FromTransaction.Type != "expense" {
		t.Errorf("Expected from transaction type 'expense', got %s", response.FromTransaction.Type)
	}
	if response.FromTransaction.Amount != "500.00" {
		t.Errorf("Expected amount '500.00', got %s", response.FromTransaction.Amount)
	}

	// Validate to transaction
	if response.ToTransaction.Type != "income" {
		t.Errorf("Expected to transaction type 'income', got %s", response.ToTransaction.Type)
	}
	if response.ToTransaction.Amount != "500.00" {
		t.Errorf("Expected amount '500.00', got %s", response.ToTransaction.Amount)
	}

	// Both should have transfer pair ID
	if response.FromTransaction.TransferPairID == nil || response.ToTransaction.TransferPairID == nil {
		t.Error("Expected both transactions to have transfer pair ID")
	}
}

func TestCreateTransfer_SameAccountError(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	reqBody := `{"fromAccountId": 1, "toAccountId": 1, "amount": "500.00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "toAccountId" {
		t.Error("Expected validation error for 'toAccountId' field")
	}
}

func TestCreateTransfer_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	reqBody := `{"fromAccountId": 1, "toAccountId": 2, "amount": "500.00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestCreateTransfer_InvalidAmount(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	reqBody := `{"fromAccountId": 1, "toAccountId": 2, "amount": "not-a-number"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateTransfer_ZeroAmount(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})
	accountRepo.AddAccount(&domain.Account{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Savings Account",
		Template:    domain.TemplateBank,
	})

	reqBody := `{"fromAccountId": 1, "toAccountId": 2, "amount": "0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "amount" {
		t.Error("Expected validation error for 'amount' field")
	}
}

func TestCreateTransfer_AccountNotFound(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	// Only add source account
	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Checking Account",
		Template:    domain.TemplateBank,
	})

	reqBody := `{"fromAccountId": 1, "toAccountId": 999, "amount": "500.00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateTransfer_MissingFromAccountId(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	reqBody := `{"toAccountId": 2, "amount": "500.00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "fromAccountId" {
		t.Error("Expected validation error for 'fromAccountId' field")
	}
}

func TestCreateTransfer_MissingToAccountId(t *testing.T) {
	e := echo.New()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, categoryRepo)
	handler := NewTransactionHandler(transactionService)

	workspaceID := int32(1)

	reqBody := `{"fromAccountId": 1, "amount": "500.00"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateTransfer(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problemDetails ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problemDetails); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problemDetails.Errors) != 1 || problemDetails.Errors[0].Field != "toAccountId" {
		t.Error("Expected validation error for 'toAccountId' field")
	}
}
