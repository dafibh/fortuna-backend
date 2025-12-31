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

func TestGetCurrent_Success(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	workspaceID := int32(1)

	// Add an account so starting balance can be calculated
	accountRepo.AddAccount(&domain.Account{
		ID:             1,
		WorkspaceID:    workspaceID,
		Name:           "Bank Account",
		AccountType:    domain.AccountTypeAsset,
		Template:       domain.TemplateBank,
		InitialBalance: decimal.NewFromInt(1000),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months/current", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetCurrent(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response MonthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	now := time.Now()
	if response.Year != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), response.Year)
	}
	if response.Month != int(now.Month()) {
		t.Errorf("Expected month %d, got %d", int(now.Month()), response.Month)
	}
	if response.StartingBalance != "1000.00" {
		t.Errorf("Expected starting balance '1000.00', got %s", response.StartingBalance)
	}
}

func TestGetCurrent_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months/current", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetCurrent(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetByYearMonth_Success(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	workspaceID := int32(1)

	// Pre-create a month
	startDate := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     workspaceID,
		Year:            2024,
		Month:           12,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: decimal.NewFromInt(5000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	// Add a transaction for December
	transactionRepo.AddTransaction(&domain.Transaction{
		ID:              1,
		WorkspaceID:     workspaceID,
		AccountID:       1,
		Name:            "Salary",
		Amount:          decimal.NewFromInt(3000),
		Type:            domain.TransactionTypeIncome,
		TransactionDate: time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
		IsPaid:          true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months/2024/12", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("year", "month")
	c.SetParamValues("2024", "12")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetByYearMonth(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response MonthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Year != 2024 {
		t.Errorf("Expected year 2024, got %d", response.Year)
	}
	if response.Month != 12 {
		t.Errorf("Expected month 12, got %d", response.Month)
	}
	if response.StartingBalance != "5000.00" {
		t.Errorf("Expected starting balance '5000.00', got %s", response.StartingBalance)
	}
	if response.TotalIncome != "3000.00" {
		t.Errorf("Expected total income '3000.00', got %s", response.TotalIncome)
	}
	if response.ClosingBalance != "8000.00" {
		t.Errorf("Expected closing balance '8000.00', got %s", response.ClosingBalance)
	}
}

func TestGetByYearMonth_InvalidYear(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	tests := []struct {
		name     string
		yearVal  string
		monthVal string
	}{
		{"Year too low", "1999", "6"},
		{"Year too high", "2101", "6"},
		{"Invalid year format", "abc", "6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/months/"+tt.yearVal+"/"+tt.monthVal, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("year", "month")
			c.SetParamValues(tt.yearVal, tt.monthVal)

			setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

			err := handler.GetByYearMonth(c)
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

			if problemDetails.Type != ErrorTypeValidation {
				t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, problemDetails.Type)
			}
		})
	}
}

func TestGetByYearMonth_InvalidMonth(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	tests := []struct {
		name     string
		yearVal  string
		monthVal string
	}{
		{"Month too low", "2025", "0"},
		{"Month too high", "2025", "13"},
		{"Invalid month format", "2025", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/months/"+tt.yearVal+"/"+tt.monthVal, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("year", "month")
			c.SetParamValues(tt.yearVal, tt.monthVal)

			setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

			err := handler.GetByYearMonth(c)
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

			if problemDetails.Type != ErrorTypeValidation {
				t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, problemDetails.Type)
			}
		})
	}
}

func TestGetByYearMonth_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months/2025/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("year", "month")
	c.SetParamValues("2025", "1")

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetByYearMonth(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetAllMonths_Success(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	workspaceID := int32(1)

	// Pre-create months
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     workspaceID,
		Year:            2024,
		Month:           11,
		StartDate:       time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2024, 11, 30, 0, 0, 0, 0, time.UTC),
		StartingBalance: decimal.NewFromInt(1000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	monthRepo.AddMonth(&domain.Month{
		ID:              2,
		WorkspaceID:     workspaceID,
		Year:            2024,
		Month:           12,
		StartDate:       time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		StartingBalance: decimal.NewFromInt(2000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetAllMonths(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []MonthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 months, got %d", len(response))
	}
}

func TestGetAllMonths_EmptyList(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetAllMonths(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []MonthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected 0 months, got %d", len(response))
	}
}

func TestGetAllMonths_WorkspaceIsolation(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	// Add month to workspace 1
	monthRepo.AddMonth(&domain.Month{
		ID:              1,
		WorkspaceID:     1,
		Year:            2024,
		Month:           12,
		StartDate:       time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		StartingBalance: decimal.NewFromInt(1000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	// Add month to workspace 2
	monthRepo.AddMonth(&domain.Month{
		ID:              2,
		WorkspaceID:     2,
		Year:            2024,
		Month:           12,
		StartDate:       time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:         time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		StartingBalance: decimal.NewFromInt(5000),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Request from workspace 1 - should only see workspace 1's month
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetAllMonths(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response []MonthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 month, got %d", len(response))
	}

	if response[0].StartingBalance != "1000.00" {
		t.Errorf("Expected workspace 1's balance '1000.00', got %s", response[0].StartingBalance)
	}
}

func TestGetAllMonths_MissingWorkspaceID(t *testing.T) {
	e := echo.New()
	monthRepo := testutil.NewMockMonthRepository()
	transactionRepo := testutil.NewMockTransactionRepository()
	accountRepo := testutil.NewMockAccountRepository()
	calcService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calcService)
	handler := NewMonthHandler(monthService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/months", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace ID set
	setupAuthContext(c, "auth0|test", "test@example.com", "Test User", "")

	err := handler.GetAllMonths(c)
	if err != nil {
		t.Fatalf("Expected JSON response, got error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}
