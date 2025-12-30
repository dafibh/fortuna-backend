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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
	transactionService := service.NewTransactionService(transactionRepo, accountRepo)
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
