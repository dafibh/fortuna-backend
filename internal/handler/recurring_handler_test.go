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

func setupRecurringHandler() (*RecurringHandler, *testutil.MockRecurringRepository, *testutil.MockAccountRepository, *testutil.MockBudgetCategoryRepository) {
	recurringRepo := testutil.NewMockRecurringRepository()
	accountRepo := testutil.NewMockAccountRepository()
	categoryRepo := testutil.NewMockBudgetCategoryRepository()
	recurringService := service.NewRecurringService(recurringRepo, accountRepo, categoryRepo)
	handler := NewRecurringHandler(recurringService)
	return handler, recurringRepo, accountRepo, categoryRepo
}

func TestCreateRecurring_Success(t *testing.T) {
	e := echo.New()
	handler, _, accountRepo, _ := setupRecurringHandler()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        "Bank Account",
	})

	reqBody := `{"name": "Rent", "amount": "1200.00", "accountId": 1, "type": "expense", "frequency": "monthly", "dueDay": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.CreateRecurring(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response RecurringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "Rent" {
		t.Errorf("Expected name 'Rent', got %s", response.Name)
	}

	if response.Amount != "1200.00" {
		t.Errorf("Expected amount '1200.00', got %s", response.Amount)
	}

	if response.Type != "expense" {
		t.Errorf("Expected type 'expense', got %s", response.Type)
	}

	if response.Frequency != "monthly" {
		t.Errorf("Expected frequency 'monthly', got %s", response.Frequency)
	}

	if response.DueDay != 1 {
		t.Errorf("Expected dueDay 1, got %d", response.DueDay)
	}

	if !response.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestCreateRecurring_ValidationError_EmptyName(t *testing.T) {
	e := echo.New()
	handler, _, accountRepo, _ := setupRecurringHandler()

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})

	reqBody := `{"name": "", "amount": "100.00", "accountId": 1, "type": "expense", "frequency": "monthly", "dueDay": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	_ = handler.CreateRecurring(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateRecurring_ValidationError_InvalidAmount(t *testing.T) {
	e := echo.New()
	handler, _, accountRepo, _ := setupRecurringHandler()

	workspaceID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          1,
		WorkspaceID: workspaceID,
	})

	reqBody := `{"name": "Test", "amount": "invalid", "accountId": 1, "type": "expense", "frequency": "monthly", "dueDay": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	_ = handler.CreateRecurring(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestGetRecurringTransactions_Success(t *testing.T) {
	e := echo.New()
	handler, recurringRepo, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetRecurringTransactions(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response RecurringListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 recurring transaction, got %d", len(response.Data))
	}

	if response.Data[0].Name != "Rent" {
		t.Errorf("Expected name 'Rent', got %s", response.Data[0].Name)
	}
}

func TestGetRecurringTransactions_FilterActive(t *testing.T) {
	e := echo.New()
	handler, recurringRepo, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Active",
		IsActive:    true,
	})
	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Inactive",
		IsActive:    false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-transactions?active=true", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetRecurringTransactions(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var response RecurringListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 active recurring transaction, got %d", len(response.Data))
	}

	if response.Data[0].Name != "Active" {
		t.Errorf("Expected 'Active', got %s", response.Data[0].Name)
	}
}

func TestGetRecurringTransaction_Success(t *testing.T) {
	e := echo.New()
	handler, recurringRepo, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Rent",
		Amount:      decimal.NewFromFloat(1200.00),
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-transactions/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.GetRecurringTransaction(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response RecurringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "Rent" {
		t.Errorf("Expected name 'Rent', got %s", response.Name)
	}
}

func TestGetRecurringTransaction_NotFound(t *testing.T) {
	e := echo.New()
	handler, _, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-transactions/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	_ = handler.GetRecurringTransaction(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateRecurring_Success(t *testing.T) {
	e := echo.New()
	handler, recurringRepo, accountRepo, _ := setupRecurringHandler()

	workspaceID := int32(1)
	accountID := int32(1)

	accountRepo.AddAccount(&domain.Account{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
		Amount:      decimal.NewFromFloat(100.00),
		AccountID:   accountID,
		Type:        domain.TransactionTypeExpense,
		Frequency:   domain.FrequencyMonthly,
		DueDay:      1,
		IsActive:    true,
	})

	reqBody := `{"name": "New Name", "amount": "200.00", "accountId": 1, "type": "expense", "frequency": "monthly", "dueDay": 15, "isActive": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/recurring-transactions/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.UpdateRecurring(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response RecurringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", response.Name)
	}

	if response.Amount != "200.00" {
		t.Errorf("Expected amount '200.00', got %s", response.Amount)
	}

	if response.DueDay != 15 {
		t.Errorf("Expected dueDay 15, got %d", response.DueDay)
	}

	if response.IsActive {
		t.Error("Expected IsActive to be false")
	}
}

func TestDeleteRecurring_Success(t *testing.T) {
	e := echo.New()
	handler, recurringRepo, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	recurringRepo.AddRecurring(&domain.RecurringTransaction{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test",
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-transactions/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	err := handler.DeleteRecurring(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestDeleteRecurring_NotFound(t *testing.T) {
	e := echo.New()
	handler, _, _, _ := setupRecurringHandler()

	workspaceID := int32(1)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-transactions/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", workspaceID)

	_ = handler.DeleteRecurring(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestCreateRecurring_NoWorkspace(t *testing.T) {
	e := echo.New()
	handler, _, _, _ := setupRecurringHandler()

	reqBody := `{"name": "Test", "amount": "100.00", "accountId": 1, "type": "expense", "frequency": "monthly", "dueDay": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-transactions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set workspace - should return 401
	_ = handler.CreateRecurring(c)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}
