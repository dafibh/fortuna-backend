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
	pool            *pgxpool.Pool
	loanRepo        domain.LoanRepository
	providerRepo    domain.LoanProviderRepository
	transactionRepo domain.TransactionRepository // v2: transactions replace loan_payments
	accountRepo     domain.AccountRepository     // v2: to look up account type for CC handling
}

// NewLoanService creates a new LoanService
func NewLoanService(pool *pgxpool.Pool, loanRepo domain.LoanRepository, providerRepo domain.LoanProviderRepository, transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository) *LoanService {
	return &LoanService{
		pool:            pool,
		loanRepo:        loanRepo,
		providerRepo:    providerRepo,
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
	}
}

// CreateLoanInput contains input for creating a loan
type CreateLoanInput struct {
	ProviderID       int32
	ItemName         string
	TotalAmount      decimal.Decimal
	NumMonths        int32
	PurchaseDate     time.Time
	InterestRate     *decimal.Decimal  // Optional override, uses provider default if nil
	Notes            *string
	PaymentAmounts   []decimal.Decimal // Optional custom amounts for each payment
	AccountID        int32             // Required: the account to use for loan payments
	SettlementIntent *string           // Optional: "immediate" or "deferred" for CC accounts
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

	// Validate account ID and get account type
	if input.AccountID <= 0 {
		return nil, domain.ErrLoanAccountInvalid
	}

	// v2: Look up account to determine if it's a CC account
	account, err := s.accountRepo.GetByID(workspaceID, input.AccountID)
	if err != nil {
		return nil, domain.ErrLoanAccountInvalid
	}

	// Determine settlement intent based on account type
	// For CC accounts: use provided intent or default to "deferred"
	// For non-CC accounts: settlement intent is not used
	var settlementIntent *string
	isCC := account.Template == domain.TemplateCreditCard
	if isCC {
		if input.SettlementIntent != nil {
			settlementIntent = input.SettlementIntent
		} else {
			defaultIntent := string(domain.SettlementIntentDeferred)
			settlementIntent = &defaultIntent
		}
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
		AccountID:         input.AccountID,
		SettlementIntent:  settlementIntent, // Use computed intent based on account type
		Notes:             input.Notes,
	}

	// Use transaction if pool is available (for transaction generation)
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

		// v2: Generate loan payment transactions instead of loan_payments
		transactions := GenerateLoanTransactions(
			workspaceID,
			createdLoan.ID,
			createdLoan.AccountID,
			createdLoan.ItemName,
			createdLoan.MonthlyPayment,
			int(createdLoan.NumMonths),
			int(createdLoan.FirstPaymentYear),
			int(createdLoan.FirstPaymentMonth),
			isCC,
			settlementIntent,
			input.PaymentAmounts,
		)

		// Create transactions in DB transaction
		if _, err := s.transactionRepo.CreateBatchTx(tx, transactions); err != nil {
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

// GetLoansByProvider retrieves all loans for a specific provider with payment statistics
// Used by item-based provider modal to display loan items with progress
func (s *LoanService) GetLoansByProvider(workspaceID int32, providerID int32) ([]*domain.LoanWithStats, error) {
	return s.loanRepo.GetByProviderWithStats(workspaceID, providerID)
}

// GetTransactionsByLoan retrieves all transactions for a specific loan
// Used by item-based provider modal to display payment months under each loan item
func (s *LoanService) GetTransactionsByLoan(workspaceID int32, loanID int32) ([]*domain.Transaction, error) {
	// First verify the loan exists and belongs to this workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}
	return s.transactionRepo.GetByLoanID(workspaceID, loanID)
}

// GetLoanByID retrieves a loan by ID within a workspace
func (s *LoanService) GetLoanByID(workspaceID int32, id int32) (*domain.Loan, error) {
	return s.loanRepo.GetByID(workspaceID, id)
}

// UpdateLoanInput contains input for updating editable loan fields
type UpdateLoanInput struct {
	ItemName   string
	Notes      *string
	ProviderID *int32 // Optional: only changeable if no payments made
}

// UpdateLoan updates the editable fields (itemName, notes, optionally provider) of a loan
// Note: Amount, months, and dates are locked after creation
// Provider can only change if no payments have been made
func (s *LoanService) UpdateLoan(workspaceID int32, id int32, input UpdateLoanInput) (*domain.Loan, error) {
	// Validate item name
	itemName := strings.TrimSpace(input.ItemName)
	if itemName == "" {
		return nil, domain.ErrLoanItemNameEmpty
	}
	if len(itemName) > 200 {
		return nil, domain.ErrLoanItemNameTooLong
	}

	// 1. Get current loan (for old provider name and comparison)
	currentLoan, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// 2. Determine final provider ID and name
	providerID := currentLoan.ProviderID
	providerChanging := false

	if input.ProviderID != nil && *input.ProviderID != currentLoan.ProviderID {
		// 3. Provider is changing - verify no paid transactions
		hasPaid, err := s.transactionRepo.HasPaidTransactionsByLoan(workspaceID, id)
		if err != nil {
			return nil, err
		}
		if hasPaid {
			return nil, domain.ErrCannotChangeProviderAfterPayments
		}
		providerID = *input.ProviderID
		providerChanging = true
	}

	// 4. Get provider for payee string (use new or current)
	provider, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		if err == domain.ErrLoanProviderNotFound {
			return nil, domain.ErrLoanProviderInvalid
		}
		return nil, err
	}

	// 5. Build new payee string: "[ProviderName] ([ItemName])"
	newPayee := provider.Name + " (" + itemName + ")"

	// 6. Update loan record
	updatedLoan, err := s.loanRepo.UpdateEditableFields(workspaceID, id, itemName, providerID, input.Notes)
	if err != nil {
		return nil, err
	}

	// 7. Cascade payee update to all transactions
	// Only cascade if item name or provider actually changed
	needsCascade := itemName != currentLoan.ItemName || providerChanging
	if needsCascade {
		if _, err := s.transactionRepo.UpdatePayeesByLoan(workspaceID, id, newPayee); err != nil {
			// Log but don't fail - loan is already updated
			// This matches the pattern in Dev Notes
			// In production, consider adding proper logging
			_ = err // silently ignore cascade errors
		}
	}

	return updatedLoan, nil
}

// LoanEditCheck contains edit eligibility information for a loan
type LoanEditCheck struct {
	CanChangeProvider    bool `json:"canChangeProvider"`
	HasPaidTransactions  bool `json:"hasPaidTransactions"`
}

// GetEditCheck returns edit eligibility for a loan (whether provider can be changed)
func (s *LoanService) GetEditCheck(workspaceID int32, id int32) (*LoanEditCheck, error) {
	// Verify loan exists
	_, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Check if any transactions are paid
	hasPaid, err := s.transactionRepo.HasPaidTransactionsByLoan(workspaceID, id)
	if err != nil {
		return nil, err
	}

	return &LoanEditCheck{
		CanChangeProvider:   !hasPaid,
		HasPaidTransactions: hasPaid,
	}, nil
}

// DeleteLoan soft-deletes a loan with cascade transaction handling
// Follows the same pattern as RecurringTemplateServiceImpl.DeleteTemplate:
// 1. Orphan paid transactions (set loan_id = NULL to keep them in history)
// 2. Hard delete unpaid transactions (future payments no longer needed)
// 3. Soft delete the loan record
func (s *LoanService) DeleteLoan(workspaceID int32, id int32) error {
	// Verify loan exists before deleting
	_, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// 1. Orphan paid transactions (keep them in history, clear loan_id)
	if err := s.transactionRepo.OrphanPaidTransactionsByLoan(workspaceID, id); err != nil {
		return err
	}

	// 2. Hard delete unpaid transactions (future payments no longer needed)
	if err := s.transactionRepo.DeleteUnpaidTransactionsByLoan(workspaceID, id); err != nil {
		return err
	}

	// 3. Soft delete the loan record
	return s.loanRepo.SoftDelete(workspaceID, id)
}

// GetDeleteStats retrieves loan and payment statistics for delete confirmation dialog
// v2: Uses transactions table via GetLoanTransactionStats
func (s *LoanService) GetDeleteStats(workspaceID int32, id int32) (*domain.Loan, *domain.LoanDeleteStats, error) {
	// Verify loan exists and belongs to workspace
	loan, err := s.loanRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, nil, err
	}

	// v2: Get transaction stats from transactions table
	txStats, err := s.transactionRepo.GetLoanTransactionStats(workspaceID, id)
	if err != nil {
		return nil, nil, err
	}

	stats := &domain.LoanDeleteStats{
		TotalCount:  txStats.PaidCount + txStats.UnpaidCount,
		PaidCount:   txStats.PaidCount,
		UnpaidCount: txStats.UnpaidCount,
		TotalAmount: txStats.PaidTotal.Add(txStats.UnpaidTotal),
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
// TODO(v2): Implement using transactions instead of loan_payments
func (s *LoanService) GetMonthlyCommitments(workspaceID int32, year, month int) (*MonthlyCommitmentsResult, error) {
	// v2 stub: Return empty result until transaction-based query is implemented
	return &MonthlyCommitmentsResult{
		Year:        year,
		Month:       month,
		TotalUnpaid: decimal.Zero,
		TotalPaid:   decimal.Zero,
		Payments:    []*domain.MonthlyPaymentDetail{},
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

// GenerateLoanTransactions creates transaction entries for each loan payment
// v2: Replaces loan_payments with transactions linked via loan_id
func GenerateLoanTransactions(
	workspaceID int32,
	loanID int32,
	accountID int32,
	itemName string,
	monthlyPayment decimal.Decimal,
	numMonths int,
	firstPaymentYear, firstPaymentMonth int,
	isCC bool,
	settlementIntent *string,
	customAmounts []decimal.Decimal,
) []*domain.Transaction {
	transactions := make([]*domain.Transaction, numMonths)
	year := firstPaymentYear
	month := firstPaymentMonth

	// Use custom amounts if provided and correct length
	useCustom := len(customAmounts) == numMonths

	// For CC accounts, convert settlement intent string to domain type
	var domainIntent *domain.SettlementIntent
	if isCC && settlementIntent != nil {
		intent := domain.SettlementIntent(*settlementIntent)
		domainIntent = &intent
	}

	for i := 0; i < numMonths; i++ {
		amount := monthlyPayment
		if useCustom {
			amount = customAmounts[i]
		}

		// Transaction date is 1st of the payment month
		transactionDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

		transactions[i] = &domain.Transaction{
			WorkspaceID:      workspaceID,
			AccountID:        accountID,
			Name:             itemName,
			Amount:           amount,
			Type:             domain.TransactionTypeExpense,
			TransactionDate:  transactionDate,
			IsPaid:           false, // Unpaid until user marks as paid
			Source:           "loan",
			LoanID:           &loanID,
			SettlementIntent: domainIntent,
		}

		// Advance to next month
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return transactions
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

	// Generate all months in range (including gaps)
	allMonths := generateMonthRange(startYear, startMonth, endYear, endMonth)

	// Fetch aggregated loan trend data from transactions
	trendData, err := s.transactionRepo.GetLoanTrendData(
		workspaceID,
		int32(startYear), int32(startMonth),
		int32(endYear), int32(endMonth),
	)
	if err != nil {
		return nil, err
	}

	// Build a map of month -> provider breakdown for quick lookup
	// Key: "YYYY-MM", Value: map of providerID -> breakdown
	monthProviderMap := make(map[string]map[int32]*domain.ProviderBreakdown)
	for _, row := range trendData {
		monthKey := formatMonth(int(row.Year), int(row.Month))
		if monthProviderMap[monthKey] == nil {
			monthProviderMap[monthKey] = make(map[int32]*domain.ProviderBreakdown)
		}
		monthProviderMap[monthKey][row.ProviderID] = &domain.ProviderBreakdown{
			ID:     row.ProviderID,
			Name:   row.ProviderName,
			Amount: row.TotalAmount,
			IsPaid: row.AllPaid,
		}
	}

	// Build result with all months (gaps will have zero amounts)
	result := make([]domain.MonthlyTrend, len(allMonths))
	for i, m := range allMonths {
		providers := []domain.ProviderBreakdown{}
		total := decimal.Zero
		allPaid := true

		if providerMap, exists := monthProviderMap[m]; exists {
			for _, breakdown := range providerMap {
				providers = append(providers, *breakdown)
				total = total.Add(breakdown.Amount)
				if !breakdown.IsPaid {
					allPaid = false
				}
			}
		}

		result[i] = domain.MonthlyTrend{
			Month:     m,
			Total:     total,
			IsPaid:    allPaid,
			Providers: providers,
		}
	}

	return &domain.TrendResponse{Months: result}, nil
}

// PayLoanMonthInput contains input for paying a loan month
type PayLoanMonthInput struct {
	LoanID int32
	Year   int
	Month  int
}

// PayLoanMonthResult contains the result of paying a loan month
type PayLoanMonthResult struct {
	SettledTransactions []*domain.Transaction
	TotalAmount         decimal.Decimal
	Message             string
}

// PayLoanMonth marks all unpaid transactions for a loan month as paid
// Works for both bank and CC transactions - CC state transitions automatically
func (s *LoanService) PayLoanMonth(workspaceID int32, input PayLoanMonthInput) (*PayLoanMonthResult, error) {
	// 1. Verify loan exists and belongs to workspace
	loan, err := s.loanRepo.GetByID(workspaceID, input.LoanID)
	if err != nil {
		return nil, err
	}

	// 2. Get unpaid transactions for this loan and month
	transactions, err := s.transactionRepo.GetLoanTransactionsByMonth(
		workspaceID, input.LoanID, input.Year, input.Month,
	)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, domain.ErrNoTransactionsToSettle
	}

	// 3. Extract IDs for bulk update
	ids := make([]int32, len(transactions))
	for i, tx := range transactions {
		ids[i] = tx.ID
	}

	// 4. Bulk mark transactions as paid (works for both bank and CC)
	// For CC transactions, this also transitions cc_state to 'settled'
	settled, err := s.transactionRepo.BulkMarkPaid(workspaceID, ids)
	if err != nil {
		return nil, err
	}

	// Verify all transactions were settled
	if len(settled) != len(ids) {
		return nil, domain.ErrLoanPaymentAtomicityFailed
	}

	// 5. Calculate total amount
	total := decimal.Zero
	for _, tx := range settled {
		total = total.Add(tx.Amount.Abs())
	}

	// 6. Format month name for message
	monthName := time.Month(input.Month).String()

	return &PayLoanMonthResult{
		SettledTransactions: settled,
		TotalAmount:         total,
		Message:             monthName + " settled for " + loan.ItemName,
	}, nil
}
