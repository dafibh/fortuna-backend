package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

// Test helper functions

func TestCalculateMonthlyPayment_ZeroInterest(t *testing.T) {
	// RM 300, 0% interest, 3 months = RM 100 per month
	total := decimal.NewFromInt(300)
	interest := decimal.Zero
	months := 3

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromInt(100)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_WithInterest(t *testing.T) {
	// RM 1000, 10% interest, 10 months
	// Total with interest: 1000 * 1.10 = 1100
	// Monthly: 1100 / 10 = 110
	total := decimal.NewFromInt(1000)
	interest := decimal.NewFromInt(10)
	months := 10

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromInt(110)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_WithDecimalInterest(t *testing.T) {
	// RM 1000, 2.5% interest, 4 months
	// Total with interest: 1000 * 1.025 = 1025
	// Monthly: 1025 / 4 = 256.25
	total := decimal.NewFromInt(1000)
	interest := decimal.NewFromFloat(2.5)
	months := 4

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromFloat(256.25)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_Rounds(t *testing.T) {
	// RM 100, 0% interest, 3 months = RM 33.33 (rounded)
	total := decimal.NewFromInt(100)
	interest := decimal.Zero
	months := 3

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.NewFromFloat(33.33)

	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s", expected.String(), result.String())
	}
}

func TestCalculateMonthlyPayment_ZeroMonths(t *testing.T) {
	total := decimal.NewFromInt(100)
	interest := decimal.Zero
	months := 0

	result := CalculateMonthlyPayment(total, interest, months)
	expected := decimal.Zero

	if !result.Equal(expected) {
		t.Errorf("Expected %s for zero months, got %s", expected.String(), result.String())
	}
}

func TestCalculateFirstPaymentMonth_BeforeCutoff(t *testing.T) {
	// Purchase on March 20, cutoff day 25 → first payment March
	purchaseDate := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 3 {
		t.Errorf("Expected 2024-03, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_OnCutoff(t *testing.T) {
	// Purchase on March 25, cutoff day 25 → first payment April
	purchaseDate := time.Date(2024, 3, 25, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_AfterCutoff(t *testing.T) {
	// Purchase on March 26, cutoff day 25 → first payment April
	purchaseDate := time.Date(2024, 3, 26, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_YearWrap(t *testing.T) {
	// Purchase on December 26, cutoff day 25 → first payment January next year
	purchaseDate := time.Date(2024, 12, 26, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2025 || month != 1 {
		t.Errorf("Expected 2025-01, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_FirstOfMonth(t *testing.T) {
	// Purchase on March 1, cutoff day 25 → first payment March
	purchaseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoffDay := 25

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 3 {
		t.Errorf("Expected 2024-03, got %d-%d", year, month)
	}
}

func TestCalculateFirstPaymentMonth_CutoffDay1(t *testing.T) {
	// Purchase on March 1, cutoff day 1 → first payment April
	purchaseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	cutoffDay := 1

	year, month := CalculateFirstPaymentMonth(purchaseDate, cutoffDay)

	if year != 2024 || month != 4 {
		t.Errorf("Expected 2024-04, got %d-%d", year, month)
	}
}

// CreateLoan tests

func TestCreateLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "iPhone Case",
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify calculated values
	expectedMonthly := decimal.NewFromInt(100) // 300 / 3
	if !loan.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), loan.MonthlyPayment.String())
	}

	if loan.FirstPaymentYear != 2024 || loan.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", loan.FirstPaymentYear, loan.FirstPaymentMonth)
	}

	if !loan.InterestRate.Equal(decimal.Zero) {
		t.Errorf("Expected interest rate 0, got %s", loan.InterestRate.String())
	}
}

func TestCreateLoan_WithInterestRateOverride(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromInt(5), // 5% default
	})

	// Override with 10%
	overrideRate := decimal.NewFromInt(10)
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Laptop",
		TotalAmount:  decimal.NewFromInt(1000),
		NumMonths:    10,
		PurchaseDate: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		InterestRate: &overrideRate,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Interest should be 10% (override), not 5% (provider default)
	if !loan.InterestRate.Equal(overrideRate) {
		t.Errorf("Expected interest rate %s, got %s", overrideRate.String(), loan.InterestRate.String())
	}

	// Monthly: 1000 * 1.10 / 10 = 110
	expectedMonthly := decimal.NewFromInt(110)
	if !loan.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), loan.MonthlyPayment.String())
	}
}

func TestCreateLoan_UsesProviderDefaultRate(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "Bank",
		CutoffDay:           15,
		DefaultInterestRate: decimal.NewFromFloat(2.5), // 2.5% default
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Phone",
		TotalAmount:  decimal.NewFromInt(1000),
		NumMonths:    4,
		PurchaseDate: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !loan.InterestRate.Equal(decimal.NewFromFloat(2.5)) {
		t.Errorf("Expected interest rate 2.5, got %s", loan.InterestRate.String())
	}
}

func TestCreateLoan_EmptyItemName(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanItemNameEmpty {
		t.Errorf("Expected ErrLoanItemNameEmpty, got %v", err)
	}
}

func TestCreateLoan_ItemNameTooLong(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	// Create a name that's 201 characters long
	longName := ""
	for i := 0; i < 201; i++ {
		longName += "A"
	}

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     longName,
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanItemNameTooLong {
		t.Errorf("Expected ErrLoanItemNameTooLong, got %v", err)
	}
}

func TestCreateLoan_ZeroAmount(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.Zero,
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanAmountInvalid {
		t.Errorf("Expected ErrLoanAmountInvalid, got %v", err)
	}
}

func TestCreateLoan_NegativeAmount(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(-100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanAmountInvalid {
		t.Errorf("Expected ErrLoanAmountInvalid, got %v", err)
	}
}

func TestCreateLoan_ZeroMonths(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    0,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanMonthsInvalid {
		t.Errorf("Expected ErrLoanMonthsInvalid, got %v", err)
	}
}

func TestCreateLoan_InvalidProvider(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   0,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_ProviderNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := CreateLoanInput{
		ProviderID:   999, // Non-existent provider
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_ProviderWrongWorkspace(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	// Provider belongs to workspace 2
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: 2,
		Name:        "Provider",
		CutoffDay:   25,
	})

	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	// Try to create in workspace 1
	_, err := service.CreateLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

func TestCreateLoan_WithNotes(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Provider",
		CutoffDay:   25,
	})

	notes := "This is a test note"
	input := CreateLoanInput{
		ProviderID:   1,
		ItemName:     "Test",
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
		Notes:        &notes,
	}

	loan, err := service.CreateLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.Notes == nil || *loan.Notes != notes {
		t.Errorf("Expected notes '%s', got %v", notes, loan.Notes)
	}
}

// PreviewLoan tests

func TestPreviewLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	providerRepo.AddLoanProvider(&domain.LoanProvider{
		ID:                  1,
		WorkspaceID:         workspaceID,
		Name:                "SPayLater",
		CutoffDay:           25,
		DefaultInterestRate: decimal.Zero,
	})

	input := PreviewLoanInput{
		ProviderID:   1,
		TotalAmount:  decimal.NewFromInt(300),
		NumMonths:    3,
		PurchaseDate: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC),
	}

	result, err := service.PreviewLoan(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMonthly := decimal.NewFromInt(100)
	if !result.MonthlyPayment.Equal(expectedMonthly) {
		t.Errorf("Expected monthly payment %s, got %s", expectedMonthly.String(), result.MonthlyPayment.String())
	}

	if result.FirstPaymentYear != 2024 || result.FirstPaymentMonth != 3 {
		t.Errorf("Expected first payment 2024-03, got %d-%d", result.FirstPaymentYear, result.FirstPaymentMonth)
	}
}

func TestPreviewLoan_InvalidProvider(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := PreviewLoanInput{
		ProviderID:   999,
		TotalAmount:  decimal.NewFromInt(100),
		NumMonths:    3,
		PurchaseDate: time.Now(),
	}

	_, err := service.PreviewLoan(1, input)
	if err != domain.ErrLoanProviderInvalid {
		t.Errorf("Expected ErrLoanProviderInvalid, got %v", err)
	}
}

// GetLoans tests

func TestGetLoans_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Loan 1",
	})
	loanRepo.AddLoan(&domain.Loan{
		ID:          2,
		WorkspaceID: workspaceID,
		ItemName:    "Loan 2",
	})

	loans, err := service.GetLoans(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans, got %d", len(loans))
	}
}

func TestGetLoans_Empty(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	loans, err := service.GetLoans(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 0 {
		t.Errorf("Expected 0 loans, got %d", len(loans))
	}
}

// GetLoanByID tests

func TestGetLoanByID_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	loan, err := service.GetLoanByID(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}
}

func TestGetLoanByID_NotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	_, err := service.GetLoanByID(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

// DeleteLoan tests

func TestDeleteLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	err := service.DeleteLoan(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify loan is deleted
	_, err = service.GetLoanByID(workspaceID, 1)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound after delete, got %v", err)
	}
}

func TestDeleteLoan_NotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	err := service.DeleteLoan(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

// GetLoansWithStats tests

func TestGetLoansWithStats_AllFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Active Loan"},
			TotalCount:       6,
			PaidCount:        2,
			RemainingBalance: decimal.NewFromInt(400),
			Progress:         33.33,
		},
		{
			Loan:             domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Completed Loan"},
			TotalCount:       3,
			PaidCount:        3,
			RemainingBalance: decimal.Zero,
			Progress:         100,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterAll)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans with 'all' filter, got %d", len(loans))
	}
}

func TestGetLoansWithStats_ActiveFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.SetActiveWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Active Loan"},
			TotalCount:       6,
			PaidCount:        2,
			RemainingBalance: decimal.NewFromInt(400),
			Progress:         33.33,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterActive)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 1 {
		t.Errorf("Expected 1 loan with 'active' filter, got %d", len(loans))
	}

	if loans[0].ItemName != "Active Loan" {
		t.Errorf("Expected 'Active Loan', got '%s'", loans[0].ItemName)
	}

	if !loans[0].RemainingBalance.GreaterThan(decimal.Zero) {
		t.Errorf("Active loan should have remaining balance > 0")
	}
}

func TestGetLoansWithStats_CompletedFilter(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.SetCompletedWithStats([]*domain.LoanWithStats{
		{
			Loan:             domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Completed Loan"},
			TotalCount:       3,
			PaidCount:        3,
			RemainingBalance: decimal.Zero,
			Progress:         100,
		},
	})

	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilterCompleted)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 1 {
		t.Errorf("Expected 1 loan with 'completed' filter, got %d", len(loans))
	}

	if loans[0].ItemName != "Completed Loan" {
		t.Errorf("Expected 'Completed Loan', got '%s'", loans[0].ItemName)
	}

	if loans[0].Progress != 100 {
		t.Errorf("Completed loan should have 100%% progress, got %.2f", loans[0].Progress)
	}
}

func TestGetLoansWithStats_EmptyResult(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	loans, err := service.GetLoansWithStats(1, domain.LoanFilterAll)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 0 {
		t.Errorf("Expected 0 loans, got %d", len(loans))
	}
}

func TestGetLoansWithStats_DefaultsToAll(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.SetLoansWithStats([]*domain.LoanWithStats{
		{Loan: domain.Loan{ID: 1, WorkspaceID: workspaceID, ItemName: "Loan 1"}},
		{Loan: domain.Loan{ID: 2, WorkspaceID: workspaceID, ItemName: "Loan 2"}},
	})

	// Empty string should default to all
	loans, err := service.GetLoansWithStats(workspaceID, domain.LoanFilter(""))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(loans) != 2 {
		t.Errorf("Expected 2 loans with empty filter (defaults to all), got %d", len(loans))
	}
}

// Domain method tests

func TestLoan_IsActive(t *testing.T) {
	tests := []struct {
		name              string
		firstPaymentYear  int32
		firstPaymentMonth int32
		numMonths         int32
		currentYear       int
		currentMonth      int
		expectedActive    bool
	}{
		{
			name:              "Active - current month is first payment month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      3,
			expectedActive:    true,
		},
		{
			name:              "Active - in the middle of payments",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      4,
			expectedActive:    true,
		},
		{
			name:              "Active - last payment month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      5,
			expectedActive:    true,
		},
		{
			name:              "Completed - one month after last payment",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			currentYear:       2024,
			currentMonth:      6,
			expectedActive:    false,
		},
		{
			name:              "Active - crosses year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4, // Nov, Dec, Jan, Feb
			currentYear:       2025,
			currentMonth:      1,
			expectedActive:    true,
		},
		{
			name:              "Completed - after year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4, // Nov, Dec, Jan, Feb
			currentYear:       2025,
			currentMonth:      3,
			expectedActive:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &domain.Loan{
				FirstPaymentYear:  tt.firstPaymentYear,
				FirstPaymentMonth: tt.firstPaymentMonth,
				NumMonths:         tt.numMonths,
			}
			result := loan.IsActive(tt.currentYear, tt.currentMonth)
			if result != tt.expectedActive {
				t.Errorf("Expected IsActive=%v, got %v", tt.expectedActive, result)
			}
		})
	}
}

// UpdateLoan tests

func TestUpdateLoan_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
	})

	notes := "Updated notes"
	input := UpdateLoanInput{
		ItemName: "Updated Name",
		Notes:    &notes,
	}

	loan, err := service.UpdateLoan(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", loan.ItemName)
	}

	if loan.Notes == nil || *loan.Notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %v", loan.Notes)
	}
}

func TestUpdateLoan_EmptyItemName(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
	})

	input := UpdateLoanInput{
		ItemName: "",
	}

	_, err := service.UpdateLoan(workspaceID, 1, input)
	if err != domain.ErrLoanItemNameEmpty {
		t.Errorf("Expected ErrLoanItemNameEmpty, got %v", err)
	}
}

func TestUpdateLoan_ItemNameTooLong(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
	})

	// Create a name that's 201 characters long
	longName := ""
	for i := 0; i < 201; i++ {
		longName += "A"
	}

	input := UpdateLoanInput{
		ItemName: longName,
	}

	_, err := service.UpdateLoan(workspaceID, 1, input)
	if err != domain.ErrLoanItemNameTooLong {
		t.Errorf("Expected ErrLoanItemNameTooLong, got %v", err)
	}
}

func TestUpdateLoan_LoanNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	input := UpdateLoanInput{
		ItemName: "New Name",
	}

	_, err := service.UpdateLoan(1, 999, input)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

func TestUpdateLoan_ClearsNotes(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, nil)

	workspaceID := int32(1)
	originalNotes := "Original notes"
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Original Name",
		Notes:       &originalNotes,
	})

	// Update with nil notes (clear notes)
	input := UpdateLoanInput{
		ItemName: "Updated Name",
		Notes:    nil,
	}

	loan, err := service.UpdateLoan(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.Notes != nil {
		t.Errorf("Expected notes to be nil, got %v", loan.Notes)
	}
}

// GetDeleteStats tests

func TestGetDeleteStats_Success(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
		TotalAmount: decimal.NewFromInt(300),
	})

	// Add some payments
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            1,
		LoanID:        1,
		PaymentNumber: 1,
		Amount:        decimal.NewFromInt(100),
		Paid:          true,
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            2,
		LoanID:        1,
		PaymentNumber: 2,
		Amount:        decimal.NewFromInt(100),
		Paid:          true,
	})
	paymentRepo.AddPayment(&domain.LoanPayment{
		ID:            3,
		LoanID:        1,
		PaymentNumber: 3,
		Amount:        decimal.NewFromInt(100),
		Paid:          false,
	})

	loan, stats, err := service.GetDeleteStats(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}

	if stats.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", stats.TotalCount)
	}

	if stats.PaidCount != 2 {
		t.Errorf("Expected paid count 2, got %d", stats.PaidCount)
	}

	if stats.UnpaidCount != 1 {
		t.Errorf("Expected unpaid count 1, got %d", stats.UnpaidCount)
	}

	expectedTotal := decimal.NewFromInt(300)
	if !stats.TotalAmount.Equal(expectedTotal) {
		t.Errorf("Expected total amount %s, got %s", expectedTotal.String(), stats.TotalAmount.String())
	}
}

func TestGetDeleteStats_LoanNotFound(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	_, _, err := service.GetDeleteStats(1, 999)
	if err != domain.ErrLoanNotFound {
		t.Errorf("Expected ErrLoanNotFound, got %v", err)
	}
}

func TestGetDeleteStats_NoPayments(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	workspaceID := int32(1)
	loanRepo.AddLoan(&domain.Loan{
		ID:          1,
		WorkspaceID: workspaceID,
		ItemName:    "Test Loan",
	})

	loan, stats, err := service.GetDeleteStats(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if loan.ItemName != "Test Loan" {
		t.Errorf("Expected 'Test Loan', got '%s'", loan.ItemName)
	}

	if stats.TotalCount != 0 {
		t.Errorf("Expected total count 0, got %d", stats.TotalCount)
	}

	if stats.PaidCount != 0 {
		t.Errorf("Expected paid count 0, got %d", stats.PaidCount)
	}
}

func TestLoan_GetLastPaymentYearMonth(t *testing.T) {
	tests := []struct {
		name              string
		firstPaymentYear  int32
		firstPaymentMonth int32
		numMonths         int32
		expectedYear      int
		expectedMonth     int
	}{
		{
			name:              "Single month",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         1,
			expectedYear:      2024,
			expectedMonth:     3,
		},
		{
			name:              "Three months same year",
			firstPaymentYear:  2024,
			firstPaymentMonth: 3,
			numMonths:         3,
			expectedYear:      2024,
			expectedMonth:     5,
		},
		{
			name:              "Crosses year boundary",
			firstPaymentYear:  2024,
			firstPaymentMonth: 11,
			numMonths:         4,
			expectedYear:      2025,
			expectedMonth:     2,
		},
		{
			name:              "12 months",
			firstPaymentYear:  2024,
			firstPaymentMonth: 1,
			numMonths:         12,
			expectedYear:      2024,
			expectedMonth:     12,
		},
		{
			name:              "24 months",
			firstPaymentYear:  2024,
			firstPaymentMonth: 1,
			numMonths:         24,
			expectedYear:      2025,
			expectedMonth:     12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &domain.Loan{
				FirstPaymentYear:  tt.firstPaymentYear,
				FirstPaymentMonth: tt.firstPaymentMonth,
				NumMonths:         tt.numMonths,
			}
			year, month := loan.GetLastPaymentYearMonth()
			if year != tt.expectedYear || month != tt.expectedMonth {
				t.Errorf("Expected %d-%02d, got %d-%02d", tt.expectedYear, tt.expectedMonth, year, month)
			}
		})
	}
}

// GetTrend tests

func TestGetTrend_WithMultipleProviders(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	workspaceID := int32(1)
	now := time.Now()
	currentYear := int32(now.Year())
	currentMonth := int32(now.Month())

	// Set up mock trend data with multiple providers
	paymentRepo.SetTrendData([]*domain.TrendRawRow{
		{
			DueYear:      currentYear,
			DueMonth:     currentMonth,
			ProviderID:   1,
			ProviderName: "Spaylater",
			Total:        decimal.NewFromInt(180),
			IsPaid:       false,
		},
		{
			DueYear:      currentYear,
			DueMonth:     currentMonth,
			ProviderID:   2,
			ProviderName: "Atome",
			Total:        decimal.NewFromInt(170),
			IsPaid:       false,
		},
	})

	result, err := service.GetTrend(workspaceID, 6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(result.Months))
	}

	// Check first month has both providers
	firstMonth := result.Months[0]
	if len(firstMonth.Providers) != 2 {
		t.Errorf("Expected 2 providers in first month, got %d", len(firstMonth.Providers))
	}

	// Check total is sum of both providers (180 + 170 = 350)
	expectedTotal := decimal.NewFromInt(350)
	if !firstMonth.Total.Equal(expectedTotal) {
		t.Errorf("Expected total %s, got %s", expectedTotal.String(), firstMonth.Total.String())
	}
}

func TestGetTrend_GapMonthHandling(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	workspaceID := int32(1)
	now := time.Now()
	currentYear := int32(now.Year())
	currentMonth := int32(now.Month())

	// Calculate month+2 (skipping one month)
	futureYear := currentYear
	futureMonth := currentMonth + 2
	if futureMonth > 12 {
		futureMonth -= 12
		futureYear++
	}

	// Set up mock data with a gap - only has data for month+2, not month+1
	paymentRepo.SetTrendData([]*domain.TrendRawRow{
		{
			DueYear:      currentYear,
			DueMonth:     currentMonth,
			ProviderID:   1,
			ProviderName: "Spaylater",
			Total:        decimal.NewFromInt(100),
			IsPaid:       true,
		},
		{
			DueYear:      futureYear,
			DueMonth:     futureMonth,
			ProviderID:   1,
			ProviderName: "Spaylater",
			Total:        decimal.NewFromInt(100),
			IsPaid:       false,
		},
	})

	result, err := service.GetTrend(workspaceID, 6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// All 6 months should be present
	if len(result.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(result.Months))
	}

	// Month 2 (index 1) should be a gap month with RM 0
	gapMonth := result.Months[1]
	if !gapMonth.Total.Equal(decimal.Zero) {
		t.Errorf("Expected gap month total to be 0, got %s", gapMonth.Total.String())
	}

	if len(gapMonth.Providers) != 0 {
		t.Errorf("Expected gap month to have no providers, got %d", len(gapMonth.Providers))
	}

	// Gap months default to isPaid=true (no payments to be unpaid)
	if !gapMonth.IsPaid {
		t.Errorf("Expected gap month isPaid to be true")
	}
}

func TestGetTrend_IsPaidCalculation(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	workspaceID := int32(1)
	now := time.Now()
	currentYear := int32(now.Year())
	currentMonth := int32(now.Month())

	// Month 1: All providers paid
	// Month 2: One provider unpaid
	nextYear := currentYear
	nextMonth := currentMonth + 1
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	paymentRepo.SetTrendData([]*domain.TrendRawRow{
		// First month - all paid
		{
			DueYear:      currentYear,
			DueMonth:     currentMonth,
			ProviderID:   1,
			ProviderName: "Spaylater",
			Total:        decimal.NewFromInt(100),
			IsPaid:       true,
		},
		{
			DueYear:      currentYear,
			DueMonth:     currentMonth,
			ProviderID:   2,
			ProviderName: "Atome",
			Total:        decimal.NewFromInt(50),
			IsPaid:       true,
		},
		// Second month - one unpaid
		{
			DueYear:      nextYear,
			DueMonth:     nextMonth,
			ProviderID:   1,
			ProviderName: "Spaylater",
			Total:        decimal.NewFromInt(100),
			IsPaid:       true,
		},
		{
			DueYear:      nextYear,
			DueMonth:     nextMonth,
			ProviderID:   2,
			ProviderName: "Atome",
			Total:        decimal.NewFromInt(50),
			IsPaid:       false, // Unpaid!
		},
	})

	result, err := service.GetTrend(workspaceID, 6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// First month should be fully paid
	if !result.Months[0].IsPaid {
		t.Errorf("Expected first month isPaid to be true (all providers paid)")
	}

	// Second month should NOT be fully paid (one provider has unpaid items)
	if result.Months[1].IsPaid {
		t.Errorf("Expected second month isPaid to be false (one provider unpaid)")
	}
}

func TestGetTrend_DefaultsTo12Months(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	// months = 0 should default to 12
	result, err := service.GetTrend(1, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 12 {
		t.Errorf("Expected 12 months (default), got %d", len(result.Months))
	}
}

func TestGetTrend_MaxLimitedTo24Months(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	// months > 24 should cap at 24
	result, err := service.GetTrend(1, 36)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 24 {
		t.Errorf("Expected 24 months (max), got %d", len(result.Months))
	}
}

func TestGetTrend_EmptyData(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	// No trend data set - should return all gap months
	result, err := service.GetTrend(1, 6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(result.Months))
	}

	// All months should be zero
	for i, month := range result.Months {
		if !month.Total.Equal(decimal.Zero) {
			t.Errorf("Month %d: expected total 0, got %s", i, month.Total.String())
		}
		if len(month.Providers) != 0 {
			t.Errorf("Month %d: expected 0 providers, got %d", i, len(month.Providers))
		}
	}
}

func TestGetTrend_MonthFormat(t *testing.T) {
	loanRepo := testutil.NewMockLoanRepository()
	providerRepo := testutil.NewMockLoanProviderRepository()
	paymentRepo := testutil.NewMockLoanPaymentRepository()
	service := NewLoanService(nil, loanRepo, providerRepo, paymentRepo)

	result, err := service.GetTrend(1, 3)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify month format is YYYY-MM
	for _, month := range result.Months {
		if len(month.Month) != 7 {
			t.Errorf("Expected month format YYYY-MM (7 chars), got %s", month.Month)
		}
		// Check format: should match YYYY-MM pattern
		if month.Month[4] != '-' {
			t.Errorf("Expected '-' at position 4, got %s", month.Month)
		}
	}
}
