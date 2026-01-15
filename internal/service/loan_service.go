package service

import (
	"context"
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// LoanService handles loan business logic
type LoanService struct {
	pool         *pgxpool.Pool
	loanRepo     domain.LoanRepository
	providerRepo domain.LoanProviderRepository
	paymentRepo  domain.LoanPaymentRepository
}

// NewLoanService creates a new LoanService
func NewLoanService(pool *pgxpool.Pool, loanRepo domain.LoanRepository, providerRepo domain.LoanProviderRepository, paymentRepo domain.LoanPaymentRepository) *LoanService {
	return &LoanService{
		pool:         pool,
		loanRepo:     loanRepo,
		providerRepo: providerRepo,
		paymentRepo:  paymentRepo,
	}
}

// CreateLoanInput contains input for creating a loan
type CreateLoanInput struct {
	ProviderID     int32
	ItemName       string
	TotalAmount    decimal.Decimal
	NumMonths      int32
	PurchaseDate   time.Time
	InterestRate   *decimal.Decimal  // Optional override, uses provider default if nil
	Notes          *string
	PaymentAmounts []decimal.Decimal // Optional custom amounts for each payment
}

// CreateLoan creates a new loan with calculated values and generates payment schedule
func (s *LoanService) CreateLoan(workspaceID int32, input CreateLoanInput) (*domain.Loan, error) {
	// Validate item name
	itemName := strings.TrimSpace(input.ItemName)
	if itemName == "" {
		return nil, domain.ErrLoanItemNameEmpty
	}
	if len(itemName) > 200 {
		return nil, domain.ErrLoanItemNameTooLong
	}

	// Validate amount
	if input.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanAmountInvalid
	}

	// Validate months
	if input.NumMonths < 1 {
		return nil, domain.ErrLoanMonthsInvalid
	}

	// Validate provider exists
	if input.ProviderID <= 0 {
		return nil, domain.ErrLoanProviderInvalid
	}

	provider, err := s.providerRepo.GetByID(workspaceID, input.ProviderID)
	if err != nil {
		if err == domain.ErrLoanProviderNotFound {
			return nil, domain.ErrLoanProviderInvalid
		}
		return nil, err
	}

	// Use provided interest rate or default from provider
	interestRate := provider.DefaultInterestRate
	if input.InterestRate != nil {
		interestRate = *input.InterestRate
	}

	// Calculate monthly payment
	monthlyPayment := CalculateMonthlyPayment(input.TotalAmount, interestRate, int(input.NumMonths))

	// Calculate first payment month based on cutoff day
	firstPaymentYear, firstPaymentMonth := CalculateFirstPaymentMonth(input.PurchaseDate, int(provider.CutoffDay))

	loan := &domain.Loan{
		WorkspaceID:       workspaceID,
		ProviderID:        input.ProviderID,
		ItemName:          itemName,
		TotalAmount:       input.TotalAmount,
		NumMonths:         input.NumMonths,
		PurchaseDate:      input.PurchaseDate,
		InterestRate:      interestRate,
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  int32(firstPaymentYear),
		FirstPaymentMonth: int32(firstPaymentMonth),
		Notes:             input.Notes,
	}

	// Use transaction if pool is available (for payment generation)
	if s.pool != nil {
		ctx := context.Background()
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		// Create loan in transaction
		createdLoan, err := s.loanRepo.CreateTx(tx, loan)
		if err != nil {
			return nil, err
		}

		// Generate payment schedule
		payments := GeneratePaymentSchedule(
			createdLoan.ID,
			createdLoan.MonthlyPayment,
			int(createdLoan.NumMonths),
			int(createdLoan.FirstPaymentYear),
			int(createdLoan.FirstPaymentMonth),
			input.PaymentAmounts,
		)

		// Create payments in transaction
		if err := s.paymentRepo.CreateBatchTx(tx, payments); err != nil {
			return nil, err
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return createdLoan, nil
	}

	// Fallback without transaction (for backwards compatibility in tests)
	return s.loanRepo.Create(loan)
}

// PreviewLoanInput contains input for previewing loan calculations
type PreviewLoanInput struct {
	ProviderID   int32
	TotalAmount  decimal.Decimal
	NumMonths    int32
	PurchaseDate time.Time
	InterestRate *decimal.Decimal // Optional override, uses provider default if nil
}

// PreviewLoanResult contains the calculated values for a loan
type PreviewLoanResult struct {
	MonthlyPayment    decimal.Decimal
	FirstPaymentYear  int
	FirstPaymentMonth int
	InterestRate      decimal.Decimal
}

// PreviewLoan calculates loan values without creating the loan
func (s *LoanService) PreviewLoan(workspaceID int32, input PreviewLoanInput) (*PreviewLoanResult, error) {
	// Validate provider exists
	if input.ProviderID <= 0 {
		return nil, domain.ErrLoanProviderInvalid
	}

	provider, err := s.providerRepo.GetByID(workspaceID, input.ProviderID)
	if err != nil {
		if err == domain.ErrLoanProviderNotFound {
			return nil, domain.ErrLoanProviderInvalid
		}
		return nil, err
	}

	// Validate amount
	if input.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanAmountInvalid
	}

	// Validate months
	if input.NumMonths < 1 {
		return nil, domain.ErrLoanMonthsInvalid
	}

	// Use provided interest rate or default from provider
	interestRate := provider.DefaultInterestRate
	if input.InterestRate != nil {
		interestRate = *input.InterestRate
	}

	// Calculate monthly payment
	monthlyPayment := CalculateMonthlyPayment(input.TotalAmount, interestRate, int(input.NumMonths))

	// Calculate first payment month based on cutoff day
	firstPaymentYear, firstPaymentMonth := CalculateFirstPaymentMonth(input.PurchaseDate, int(provider.CutoffDay))

	return &PreviewLoanResult{
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  firstPaymentYear,
		FirstPaymentMonth: firstPaymentMonth,
		InterestRate:      interestRate,
	}, nil
}

// GetLoans retrieves all loans for a workspace
func (s *LoanService) GetLoans(workspaceID int32) ([]*domain.Loan, error) {
	return s.loanRepo.GetAllByWorkspace(workspaceID)
}

// GetActiveLoans retrieves active loans for a workspace
func (s *LoanService) GetActiveLoans(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	return s.loanRepo.GetActiveByWorkspace(workspaceID, currentYear, currentMonth)
}

// GetCompletedLoans retrieves completed loans for a workspace
func (s *LoanService) GetCompletedLoans(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	return s.loanRepo.GetCompletedByWorkspace(workspaceID, currentYear, currentMonth)
}

// GetLoansWithStats retrieves loans with payment statistics based on filter
func (s *LoanService) GetLoansWithStats(workspaceID int32, filter domain.LoanFilter) ([]*domain.LoanWithStats, error) {
	switch filter {
	case domain.LoanFilterActive:
		return s.loanRepo.GetActiveWithStats(workspaceID)
	case domain.LoanFilterCompleted:
		return s.loanRepo.GetCompletedWithStats(workspaceID)
	default:
		return s.loanRepo.GetAllWithStats(workspaceID)
	}
}

// GetLoanByID retrieves a loan by ID within a workspace
func (s *LoanService) GetLoanByID(workspaceID int32, id int32) (*domain.Loan, error) {
	return s.loanRepo.GetByID(workspaceID, id)
}

// UpdateLoanInput contains input for updating editable loan fields
type UpdateLoanInput struct {
	ItemName string
	Notes    *string
}

// UpdateLoan updates the editable fields (itemName, notes) of a loan
// Note: Amount, months, and dates are locked after creation
func (s *LoanService) UpdateLoan(workspaceID int32, id int32, input UpdateLoanInput) (*domain.Loan, error) {
	// Validate item name
	itemName := strings.TrimSpace(input.ItemName)
	if itemName == "" {
		return nil, domain.ErrLoanItemNameEmpty
	}
	if len(itemName) > 200 {
		return nil, domain.ErrLoanItemNameTooLong
	}

	// Verify loan exists
	_, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	return s.loanRepo.UpdatePartial(workspaceID, id, itemName, input.Notes)
}

// DeleteLoan soft-deletes a loan
func (s *LoanService) DeleteLoan(workspaceID int32, id int32) error {
	// Verify loan exists before deleting
	_, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}
	return s.loanRepo.SoftDelete(workspaceID, id)
}

// GetDeleteStats retrieves loan and payment statistics for delete confirmation dialog
func (s *LoanService) GetDeleteStats(workspaceID int32, id int32) (*domain.Loan, *domain.LoanDeleteStats, error) {
	// Verify loan exists and belongs to workspace
	loan, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, nil, err
	}

	// Get payment statistics
	stats, err := s.paymentRepo.GetDeleteStats(id)
	if err != nil {
		return nil, nil, err
	}

	return loan, stats, nil
}

// MonthlyCommitmentsResult contains aggregated loan commitments for a month
type MonthlyCommitmentsResult struct {
	Year        int
	Month       int
	TotalUnpaid decimal.Decimal
	TotalPaid   decimal.Decimal
	Payments    []*domain.MonthlyPaymentDetail
}

// GetMonthlyCommitments retrieves loan commitments for a specific month
func (s *LoanService) GetMonthlyCommitments(workspaceID int32, year, month int) (*MonthlyCommitmentsResult, error) {
	// Get all payments with loan details for the month
	payments, err := s.paymentRepo.GetPaymentsWithDetailsByMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	totalUnpaid := decimal.Zero
	totalPaid := decimal.Zero
	for _, p := range payments {
		if p.Paid {
			totalPaid = totalPaid.Add(p.Amount)
		} else {
			totalUnpaid = totalUnpaid.Add(p.Amount)
		}
	}

	return &MonthlyCommitmentsResult{
		Year:        year,
		Month:       month,
		TotalUnpaid: totalUnpaid,
		TotalPaid:   totalPaid,
		Payments:    payments,
	}, nil
}

// CalculateMonthlyPayment calculates the monthly payment for a loan
// Formula: (totalAmount * (1 + interestRate/100)) / numMonths
func CalculateMonthlyPayment(totalAmount, interestRate decimal.Decimal, numMonths int) decimal.Decimal {
	if numMonths <= 0 {
		return decimal.Zero
	}
	multiplier := decimal.NewFromInt(1).Add(interestRate.Div(decimal.NewFromInt(100)))
	totalWithInterest := totalAmount.Mul(multiplier)
	return totalWithInterest.Div(decimal.NewFromInt(int64(numMonths))).Round(2)
}

// CalculateFirstPaymentMonth calculates the first payment year and month based on purchase date and cutoff day
// If purchase day < cutoff day → first payment in current month
// If purchase day >= cutoff day → first payment in next month
func CalculateFirstPaymentMonth(purchaseDate time.Time, cutoffDay int) (year, month int) {
	if purchaseDate.Day() < cutoffDay {
		return purchaseDate.Year(), int(purchaseDate.Month())
	}
	// Next month
	nextMonth := purchaseDate.AddDate(0, 1, 0)
	return nextMonth.Year(), int(nextMonth.Month())
}

// GeneratePaymentSchedule generates all payment entries for a loan
// If customAmounts is provided and matches numMonths, use those amounts instead of monthlyPayment
func GeneratePaymentSchedule(loanID int32, monthlyPayment decimal.Decimal, numMonths int, firstPaymentYear, firstPaymentMonth int, customAmounts []decimal.Decimal) []*domain.LoanPayment {
	payments := make([]*domain.LoanPayment, numMonths)
	year := firstPaymentYear
	month := firstPaymentMonth

	// Use custom amounts if provided and correct length
	useCustom := len(customAmounts) == numMonths

	for i := 0; i < numMonths; i++ {
		amount := monthlyPayment
		if useCustom {
			amount = customAmounts[i]
		}

		payments[i] = &domain.LoanPayment{
			LoanID:        loanID,
			PaymentNumber: int32(i + 1), // 1-indexed
			Amount:        amount,
			DueYear:       int32(year),
			DueMonth:      int32(month),
			Paid:          false,
		}

		// Advance to next month
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return payments
}

// GetTrend retrieves trend data for loan payments aggregated by month
// Returns monthly totals with provider breakdown for the specified number of months
// Starting from current month, includes gap months with RM 0.00
func (s *LoanService) GetTrend(workspaceID int32, months int) (*domain.TrendResponse, error) {
	// Validate and apply defaults
	if months <= 0 {
		months = 12
	}
	if months > 24 {
		months = 24
	}

	// Get current year/month as start
	now := time.Now()
	startYear := now.Year()
	startMonth := int(now.Month())

	// Calculate end year/month
	endYear, endMonth := startYear, startMonth
	for i := 1; i < months; i++ {
		endYear, endMonth = nextMonth(endYear, endMonth)
	}

	// Fetch raw data from repository
	rawData, err := s.paymentRepo.GetTrendRaw(workspaceID, int32(startYear), int32(startMonth))
	if err != nil {
		return nil, err
	}

	// Generate all months in range (including gaps)
	allMonths := generateMonthRange(startYear, startMonth, endYear, endMonth)

	// Create a map for quick lookup: "YYYY-MM" -> month data
	monthMap := make(map[string]*domain.MonthlyTrend)
	for _, m := range allMonths {
		monthMap[m] = &domain.MonthlyTrend{
			Month:     m,
			Total:     decimal.Zero,
			IsPaid:    true, // Defaults to true, will be set to false if any unpaid found
			Providers: []domain.ProviderBreakdown{},
		}
	}

	// Process raw data - aggregate by month and provider
	for _, row := range rawData {
		monthKey := formatMonth(int(row.DueYear), int(row.DueMonth))

		// Only process months within our range
		monthData, exists := monthMap[monthKey]
		if !exists {
			continue
		}

		// Add provider breakdown
		monthData.Providers = append(monthData.Providers, domain.ProviderBreakdown{
			ID:     row.ProviderID,
			Name:   row.ProviderName,
			Amount: row.Total,
		})

		// Add to total
		monthData.Total = monthData.Total.Add(row.Total)

		// If any provider has unpaid items, the month is not fully paid
		if !row.IsPaid {
			monthData.IsPaid = false
		}
	}

	// Convert map to ordered slice
	result := make([]domain.MonthlyTrend, len(allMonths))
	for i, m := range allMonths {
		result[i] = *monthMap[m]
	}

	return &domain.TrendResponse{Months: result}, nil
}
