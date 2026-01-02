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

func TestCreateLoanProvider_Success(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "Bank ABC", "cutoffDay": 15, "defaultInterestRate": "1.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-providers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var response LoanProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "Bank ABC" {
		t.Errorf("Expected name 'Bank ABC', got %s", response.Name)
	}

	if response.CutoffDay != 15 {
		t.Errorf("Expected cutoff day 15, got %d", response.CutoffDay)
	}

	if response.DefaultInterestRate != "1.50" {
		t.Errorf("Expected interest rate '1.50', got %s", response.DefaultInterestRate)
	}
}

func TestCreateLoanProvider_EmptyName(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "", "cutoffDay": 15, "defaultInterestRate": "1.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-providers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var problem ProblemDetails
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(problem.Errors) == 0 {
		t.Error("Expected validation errors in response")
	}
}

func TestCreateLoanProvider_InvalidCutoffDay(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "Bank Test", "cutoffDay": 32, "defaultInterestRate": "1.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-providers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateLoanProvider_InvalidInterestRate(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "Bank Test", "cutoffDay": 15, "defaultInterestRate": "-1.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-providers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.CreateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestCreateLoanProvider_NoWorkspace(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "Bank ABC", "cutoffDay": 15, "defaultInterestRate": "1.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-providers", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No workspace set (workspaceID = 0)
	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 0)

	err := handler.CreateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestGetLoanProviders_Success(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	// Add some test providers
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Bank ABC",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  2,
		WorkspaceID:         1,
		Name:                "Bank XYZ",
		CutoffDay:           20,
		DefaultInterestRate: decimal.NewFromFloat(2.0),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoanProviders(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []LoanProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(response))
	}
}

func TestGetLoanProviders_EmptyList(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-providers", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoanProviders(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response []LoanProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(response))
	}
}

func TestGetLoanProvider_Success(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Bank ABC",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-providers/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "Bank ABC" {
		t.Errorf("Expected name 'Bank ABC', got %s", response.Name)
	}
}

func TestGetLoanProvider_NotFound(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-providers/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.GetLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestUpdateLoanProvider_Success(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Old Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})

	reqBody := `{"name": "New Name", "cutoffDay": 20, "defaultInterestRate": "2.50"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loan-providers/1", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response LoanProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", response.Name)
	}

	if response.CutoffDay != 20 {
		t.Errorf("Expected cutoff day 20, got %d", response.CutoffDay)
	}

	if response.DefaultInterestRate != "2.50" {
		t.Errorf("Expected interest rate '2.50', got %s", response.DefaultInterestRate)
	}
}

func TestUpdateLoanProvider_NotFound(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	reqBody := `{"name": "New Name", "cutoffDay": 20, "defaultInterestRate": "2.50"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/loan-providers/999", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.UpdateLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestDeleteLoanProvider_Success(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Bank ABC",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/loan-providers/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.DeleteLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rec.Code)
	}
}

func TestDeleteLoanProvider_NotFound(t *testing.T) {
	e := echo.New()
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := service.NewLoanProviderService(providerRepo)
	handler := NewLoanProviderHandler(providerService)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/loan-providers/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	setupAuthContextWithWorkspace(c, "auth0|test", "test@example.com", "Test User", "", 1)

	err := handler.DeleteLoanProvider(c)
	if err != nil {
		t.Fatalf("Expected no error (error should be in response), got %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}
