package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

// CreateProvider tests

func TestCreateProvider_Success(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Bank ABC",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	}

	provider, err := providerService.CreateProvider(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.Name != "Bank ABC" {
		t.Errorf("Expected name 'Bank ABC', got %s", provider.Name)
	}

	if provider.CutoffDay != 15 {
		t.Errorf("Expected cutoff day 15, got %d", provider.CutoffDay)
	}

	if !provider.DefaultInterestRate.Equal(decimal.NewFromFloat(1.5)) {
		t.Errorf("Expected interest rate 1.5, got %s", provider.DefaultInterestRate.String())
	}

	if provider.WorkspaceID != workspaceID {
		t.Errorf("Expected workspace ID %d, got %d", workspaceID, provider.WorkspaceID)
	}
}

func TestCreateProvider_TrimsName(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "  Bank XYZ  ",
		CutoffDay:           1,
		DefaultInterestRate: decimal.Zero,
	}

	provider, err := providerService.CreateProvider(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.Name != "Bank XYZ" {
		t.Errorf("Expected trimmed name 'Bank XYZ', got '%s'", provider.Name)
	}
}

func TestCreateProvider_EmptyName(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}

	if err != domain.ErrLoanProviderNameEmpty {
		t.Errorf("Expected ErrLoanProviderNameEmpty, got %v", err)
	}
}

func TestCreateProvider_WhitespaceOnlyName(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "   ",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for whitespace-only name, got nil")
	}

	if err != domain.ErrLoanProviderNameEmpty {
		t.Errorf("Expected ErrLoanProviderNameEmpty, got %v", err)
	}
}

func TestCreateProvider_InvalidCutoffDay_Zero(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Bank Test",
		CutoffDay:           0,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for cutoff day 0, got nil")
	}

	if err != domain.ErrInvalidCutoffDay {
		t.Errorf("Expected ErrInvalidCutoffDay, got %v", err)
	}
}

func TestCreateProvider_InvalidCutoffDay_Above31(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Bank Test",
		CutoffDay:           32,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for cutoff day 32, got nil")
	}

	if err != domain.ErrInvalidCutoffDay {
		t.Errorf("Expected ErrInvalidCutoffDay, got %v", err)
	}
}

func TestCreateProvider_ValidCutoffDay_Boundaries(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	// Test cutoff day 1 (minimum valid)
	input1 := CreateProviderInput{
		Name:                "Bank Min",
		CutoffDay:           1,
		DefaultInterestRate: decimal.Zero,
	}
	provider1, err := providerService.CreateProvider(workspaceID, input1)
	if err != nil {
		t.Fatalf("Expected no error for cutoff day 1, got %v", err)
	}
	if provider1.CutoffDay != 1 {
		t.Errorf("Expected cutoff day 1, got %d", provider1.CutoffDay)
	}

	// Test cutoff day 31 (maximum valid)
	input31 := CreateProviderInput{
		Name:                "Bank Max",
		CutoffDay:           31,
		DefaultInterestRate: decimal.Zero,
	}
	provider31, err := providerService.CreateProvider(workspaceID, input31)
	if err != nil {
		t.Fatalf("Expected no error for cutoff day 31, got %v", err)
	}
	if provider31.CutoffDay != 31 {
		t.Errorf("Expected cutoff day 31, got %d", provider31.CutoffDay)
	}
}

func TestCreateProvider_InvalidInterestRate_Negative(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Bank Test",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(-0.5),
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for negative interest rate, got nil")
	}

	if err != domain.ErrInvalidInterestRate {
		t.Errorf("Expected ErrInvalidInterestRate, got %v", err)
	}
}

func TestCreateProvider_ZeroInterestRate(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Bank Zero Rate",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	provider, err := providerService.CreateProvider(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error for zero interest rate, got %v", err)
	}

	if !provider.DefaultInterestRate.IsZero() {
		t.Errorf("Expected zero interest rate, got %s", provider.DefaultInterestRate.String())
	}
}

func TestCreateProvider_NameTooLong(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	// Create a name that's 101 characters long
	longName := "A"
	for i := 0; i < 100; i++ {
		longName += "A"
	}
	input := CreateProviderInput{
		Name:                longName,
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for name > 100 characters, got nil")
	}

	if err != domain.ErrLoanProviderNameTooLong {
		t.Errorf("Expected ErrLoanProviderNameTooLong, got %v", err)
	}
}

func TestCreateProvider_NameExactly100Characters(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	// Create a name that's exactly 100 characters
	name100 := ""
	for i := 0; i < 100; i++ {
		name100 += "A"
	}
	input := CreateProviderInput{
		Name:                name100,
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	provider, err := providerService.CreateProvider(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error for name = 100 characters, got %v", err)
	}

	if len(provider.Name) != 100 {
		t.Errorf("Expected name length 100, got %d", len(provider.Name))
	}
}

func TestCreateProvider_InterestRateTooHigh(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "High Rate Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(100.01),
	}

	_, err := providerService.CreateProvider(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for interest rate > 100%, got nil")
	}

	if err != domain.ErrInterestRateTooHigh {
		t.Errorf("Expected ErrInterestRateTooHigh, got %v", err)
	}
}

func TestCreateProvider_InterestRateExactly100(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	input := CreateProviderInput{
		Name:                "Max Rate Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromInt(100),
	}

	provider, err := providerService.CreateProvider(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error for interest rate = 100%%, got %v", err)
	}

	if !provider.DefaultInterestRate.Equal(decimal.NewFromInt(100)) {
		t.Errorf("Expected interest rate 100, got %s", provider.DefaultInterestRate.String())
	}
}

// GetProviders tests

func TestGetProviders_Success(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	// Add some providers
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Provider 1",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  2,
		WorkspaceID:         workspaceID,
		Name:                "Provider 2",
		CutoffDay:           20,
		DefaultInterestRate: decimal.NewFromFloat(2.0),
	})

	providers, err := providerService.GetProviders(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestGetProviders_EmptyList(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	providers, err := providerService.GetProviders(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestGetProviders_WorkspaceIsolation(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	// Add provider to workspace 1
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Provider WS1",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// Add provider to workspace 2
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  2,
		WorkspaceID:         2,
		Name:                "Provider WS2",
		CutoffDay:           20,
		DefaultInterestRate: decimal.Zero,
	})

	// Query workspace 1 - should only see 1 provider
	providers1, err := providerService.GetProviders(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(providers1) != 1 {
		t.Errorf("Expected 1 provider for workspace 1, got %d", len(providers1))
	}
	if providers1[0].Name != "Provider WS1" {
		t.Errorf("Expected 'Provider WS1', got %s", providers1[0].Name)
	}

	// Query workspace 2 - should only see 1 provider
	providers2, err := providerService.GetProviders(2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(providers2) != 1 {
		t.Errorf("Expected 1 provider for workspace 2, got %d", len(providers2))
	}
	if providers2[0].Name != "Provider WS2" {
		t.Errorf("Expected 'Provider WS2', got %s", providers2[0].Name)
	}
}

// GetProviderByID tests

func TestGetProviderByID_Success(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerID := int32(1)

	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  providerID,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.5),
	})

	provider, err := providerService.GetProviderByID(workspaceID, providerID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.Name != "Test Provider" {
		t.Errorf("Expected name 'Test Provider', got %s", provider.Name)
	}
}

func TestGetProviderByID_NotFound(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	_, err := providerService.GetProviderByID(workspaceID, 999)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound, got %v", err)
	}
}

func TestGetProviderByID_WrongWorkspace(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	// Provider belongs to workspace 1
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// Try to get it from workspace 2
	_, err := providerService.GetProviderByID(2, 1)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound for wrong workspace, got %v", err)
	}
}

// UpdateProvider tests

func TestUpdateProvider_Success(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Old Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(1.0),
	})

	input := UpdateProviderInput{
		Name:                "New Name",
		CutoffDay:           20,
		DefaultInterestRate: decimal.NewFromFloat(2.5),
	}

	provider, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", provider.Name)
	}

	if provider.CutoffDay != 20 {
		t.Errorf("Expected cutoff day 20, got %d", provider.CutoffDay)
	}

	if !provider.DefaultInterestRate.Equal(decimal.NewFromFloat(2.5)) {
		t.Errorf("Expected interest rate 2.5, got %s", provider.DefaultInterestRate.String())
	}
}

func TestUpdateProvider_TrimsName(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Old Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "  New Name  ",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	provider, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.Name != "New Name" {
		t.Errorf("Expected trimmed name 'New Name', got '%s'", provider.Name)
	}
}

func TestUpdateProvider_EmptyName(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Old Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err != domain.ErrLoanProviderNameEmpty {
		t.Errorf("Expected ErrLoanProviderNameEmpty, got %v", err)
	}
}

func TestUpdateProvider_InvalidCutoffDay(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "Test Provider",
		CutoffDay:           32,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err != domain.ErrInvalidCutoffDay {
		t.Errorf("Expected ErrInvalidCutoffDay, got %v", err)
	}
}

func TestUpdateProvider_InvalidInterestRate(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(-1.0),
	}

	_, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err != domain.ErrInvalidInterestRate {
		t.Errorf("Expected ErrInvalidInterestRate, got %v", err)
	}
}

func TestUpdateProvider_NameTooLong(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// Create a name that's 101 characters long
	longName := "A"
	for i := 0; i < 100; i++ {
		longName += "A"
	}
	input := UpdateProviderInput{
		Name:                longName,
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err == nil {
		t.Fatal("Expected error for name > 100 characters, got nil")
	}

	if err != domain.ErrLoanProviderNameTooLong {
		t.Errorf("Expected ErrLoanProviderNameTooLong, got %v", err)
	}
}

func TestUpdateProvider_InterestRateTooHigh(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(100.01),
	}

	_, err := providerService.UpdateProvider(workspaceID, 1, input)
	if err == nil {
		t.Fatal("Expected error for interest rate > 100%, got nil")
	}

	if err != domain.ErrInterestRateTooHigh {
		t.Errorf("Expected ErrInterestRateTooHigh, got %v", err)
	}
}

func TestUpdateProvider_NotFound(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	input := UpdateProviderInput{
		Name:                "New Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	_, err := providerService.UpdateProvider(workspaceID, 999, input)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound, got %v", err)
	}
}

func TestUpdateProvider_WrongWorkspace(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	// Provider belongs to workspace 1
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	input := UpdateProviderInput{
		Name:                "New Name",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	}

	// Try to update it from workspace 2
	_, err := providerService.UpdateProvider(2, 1, input)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound for wrong workspace, got %v", err)
	}
}

// DeleteProvider tests

func TestDeleteProvider_Success(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	err := providerService.DeleteProvider(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify provider is soft-deleted (not found when querying)
	_, err = providerService.GetProviderByID(workspaceID, 1)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound after soft delete, got %v", err)
	}
}

func TestDeleteProvider_NotFound(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)

	err := providerService.DeleteProvider(workspaceID, 999)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound, got %v", err)
	}
}

func TestDeleteProvider_WrongWorkspace(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	// Provider belongs to workspace 1
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         1,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// Try to delete it from workspace 2
	err := providerService.DeleteProvider(2, 1)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound for wrong workspace, got %v", err)
	}
}

func TestDeleteProvider_AlreadyDeleted(t *testing.T) {
	providerRepo := testutil.NewMockLoanProviderRepository()
	providerService := NewLoanProviderService(providerRepo)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Test Provider",
		CutoffDay:           15,
		DefaultInterestRate: decimal.Zero,
	})

	// First delete should succeed
	err := providerService.DeleteProvider(workspaceID, 1)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Second delete should fail (already deleted)
	err = providerService.DeleteProvider(workspaceID, 1)
	if err != domain.ErrLoanProviderNotFound {
		t.Errorf("Expected ErrLoanProviderNotFound for already deleted provider, got %v", err)
	}
}
