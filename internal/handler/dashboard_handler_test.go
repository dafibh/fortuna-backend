package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

func TestGetSummary_Success(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	workspaceID := int32(1)

	// Add an account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Add income transaction
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(5000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: time.Now(),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response DashboardSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify total balance includes account + income
	if response.TotalBalance != "15000.00" {
		t.Errorf("Expected total balance '15000.00', got %s", response.TotalBalance)
	}

	// Verify month data is present
	if response.Month.Year == 0 {
		t.Error("Expected month year to be set")
	}
}

func TestGetSummary_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetSummary_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	// Add account to workspace 1
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    1,
		Name:           "Bank WS1",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Add account to workspace 2
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    2,
		Name:           "Bank WS2",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(5000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Request from workspace 1 - should only see workspace 1's data
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response DashboardSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should see workspace 1's balance (10000), not workspace 2's
	if response.TotalBalance != "10000.00" {
		t.Errorf("Expected workspace 1's balance '10000.00', got %s", response.TotalBalance)
	}
}

func TestGetSummary_WithLiabilities(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	workspaceID := int32(1)

	// Add asset account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(10000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Add liability account (credit card)
	accountRepo.AddAccount(&domain.Account{
		ID:             2,
		WorkspaceID:    workspaceID,
		Name:           "Credit Card",
		AccountType:    domain.AccountTypeLiability,
		Template:       domain.TemplateCreditCard,
		InitialBalance: decimal.Zero,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Add CC expense (creates debt)
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       2,
		Name:            "Shopping",
		Amount:          decimal.NewFromInt(1000),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: time.Now(),
		IsPaid:          false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response DashboardSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Total should be 10000 (bank) - 1000 (CC debt) = 9000
	if response.TotalBalance != "9000.00" {
		t.Errorf("Expected total balance '9000.00' (asset - liability), got %s", response.TotalBalance)
	}
}

func TestGetSummary_ReturnsValidResponse(t *testing.T) {
	// Note: Detailed calculation tests are in dashboard_service_test.go
	// This handler test verifies the API returns a valid response structure
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	workspaceID := int32(1)
	now := time.Now()
	txDate := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, time.UTC)

	// Add account
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(5000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	// Add transactions
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(3000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              2,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Groceries",
		Amount:          decimal.NewFromInt(500),
		Type:            domain.TransactionTypeExpense,
		TransactionDate: txDate,
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response DashboardSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure has all required fields
	if response.TotalBalance == "" {
		t.Error("Expected TotalBalance to be set")
	}
	if response.InHandBalance == "" {
		t.Error("Expected InHandBalance to be set")
	}
	if response.Month.Year != now.Year() {
		t.Errorf("Expected month year %d, got %d", now.Year(), response.Month.Year)
	}
	if response.Month.Month != int(now.Month()) {
		t.Errorf("Expected month %d, got %d", int(now.Month()), response.Month.Month)
	}
}

func TestGetSummary_EmptyAccounts(t *testing.T) {
	e := echo.New()
	accountRepo := testutil.NewMockAccountRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	monthRepo := testutil.NewMockMonthRepository()
	loanPaymentRepo := testutil.NewMockLoanPaymentRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, loanPaymentRepo, monthService, calcService)
	handler := NewDashboardHandler(dashboardService)

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetSummary(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response DashboardSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should return zero balances
	if response.TotalBalance != "0.00" {
		t.Errorf("Expected total balance '0.00', got %s", response.TotalBalance)
	}
	if response.InHandBalance != "0.00" {
		t.Errorf("Expected in-hand balance '0.00', got %s", response.InHandBalance)
	}
}
