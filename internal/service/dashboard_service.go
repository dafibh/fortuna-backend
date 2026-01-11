package service

import (
	"errors"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// ErrProjectionLimitExceeded is returned when requesting projections beyond the maximum allowed
var ErrProjectionLimitExceeded = errors.New("projection limit exceeded (max 12 months)")

// DashboardService handles dashboard-related business logic
type DashboardService struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
	loanPaymentRepo domain.LoanPaymentRepository
	monthService    *MonthService
	calcService     *CalculationService
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(
	accountRepo domain.AccountRepository,
	transactionRepo domain.TransactionRepository,
	loanPaymentRepo domain.LoanPaymentRepository,
	monthService *MonthService,
	calcService *CalculationService,
) *DashboardService {
	return &DashboardService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		loanPaymentRepo: loanPaymentRepo,
		monthService:    monthService,
		calcService:     calcService,
	}
}

// GetSummary returns the dashboard summary for a workspace for the current month
func (s *DashboardService) GetSummary(workspaceID int32) (*domain.DashboardSummary, error) {
	now := time.Now()
	return s.GetSummaryForMonth(workspaceID, now.Year(), int(now.Month()))
}

// GetSummaryForMonth returns the dashboard summary for a workspace for a specific month
// For future months, it returns projected data based on current balances
// NOTE: Uses server's local timezone for month boundary detection. Consider accepting
// timezone from request if users report unexpected behavior near month boundaries.
func (s *DashboardService) GetSummaryForMonth(workspaceID int32, year, month int) (*domain.DashboardSummary, error) {
	now := time.Now()
	requestedDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	isFuture := requestedDate.After(currentMonthStart)

	// Check projection limit for future months
	if isFuture {
		monthsAhead := monthsBetween(currentMonthStart, requestedDate)
		if monthsAhead > domain.MaxProjectionMonths {
			return nil, ErrProjectionLimitExceeded
		}
		return s.getProjection(workspaceID, year, month)
	}

	return s.getActualSummary(workspaceID, year, month)
}

// getActualSummary returns the dashboard summary for current or past months
func (s *DashboardService) getActualSummary(workspaceID int32, year, month int) (*domain.DashboardSummary, error) {
	// 1. Get month data
	monthData, err := s.monthService.GetOrCreateMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// 2. Calculate total balance from all accounts (assets - liabilities)
	totalBalance, err := s.calculateTotalBalance(workspaceID)
	if err != nil {
		return nil, err
	}

	// 3. Calculate in-hand balance (starting + income - paid expenses only)
	paidExpenses, err := s.transactionRepo.SumPaidExpensesByDateRange(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	inHandBalance := monthData.StartingBalance.Add(monthData.TotalIncome).Sub(paidExpenses)

	// 4. Calculate unpaid expenses for disposable income
	// Use SumUnpaidExpensesForDisposable which EXCLUDES deferred CC transactions
	// (deferred CC transactions are obligations for next month, not this month)
	unpaidExpenses, err := s.transactionRepo.SumUnpaidExpensesForDisposable(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	// 5. Calculate unpaid loan payments for the month
	unpaidLoanPayments, err := s.loanPaymentRepo.SumUnpaidByMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Disposable = In-Hand - Unpaid Expenses - Unpaid Loan Payments
	disposableIncome := inHandBalance.Sub(unpaidExpenses).Sub(unpaidLoanPayments)

	// 6. Calculate days remaining in the month
	daysRemaining := s.calculateDaysRemaining(year, month)

	// 7. Calculate daily budget
	dailyBudget := decimal.Zero
	if daysRemaining > 0 {
		dailyBudget = disposableIncome.Div(decimal.NewFromInt(int64(daysRemaining)))
	}

	return &domain.DashboardSummary{
		IsProjection:       false,
		TotalBalance:       totalBalance,
		InHandBalance:      inHandBalance,
		DisposableIncome:   disposableIncome,
		UnpaidExpenses:     unpaidExpenses,
		UnpaidLoanPayments: unpaidLoanPayments,
		DaysRemaining:      daysRemaining,
		DailyBudget:        dailyBudget,
		Month:              monthData,
	}, nil
}

// getProjection returns projected dashboard data for a future month
func (s *DashboardService) getProjection(workspaceID int32, year, month int) (*domain.DashboardSummary, error) {
	// Get the starting balance by chaining from current month
	startingBalance, err := s.chainBalanceToMonth(workspaceID)
	if err != nil {
		return nil, err
	}

	// Get loan payments due for this future month
	unpaidLoanPayments, err := s.loanPaymentRepo.SumUnpaidByMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}

	// Get deferred CC from previous month(s) - these are obligations for this projected month
	// Calculate the previous month's date range
	prevMonthDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
	prevMonthStart := time.Date(prevMonthDate.Year(), prevMonthDate.Month(), 1, 0, 0, 0, 0, time.Local)
	prevMonthEnd := time.Date(prevMonthDate.Year(), prevMonthDate.Month()+1, 0, 0, 0, 0, 0, time.Local)

	deferredFromPrevMonth, err := s.transactionRepo.SumDeferredCCByDateRange(
		workspaceID, prevMonthStart, prevMonthEnd)
	if err != nil {
		return nil, err
	}

	// Build projection details with loan payments and deferred CC
	projection := &domain.ProjectionDetails{
		RecurringIncome:   decimal.Zero,
		RecurringExpenses: decimal.Zero,
		LoanPayments:      unpaidLoanPayments,
		Note:              "Recurring transactions not yet configured",
	}

	// Total obligations = loan payments + deferred CC from previous month
	totalObligations := unpaidLoanPayments.Add(deferredFromPrevMonth)

	// Projected closing = starting - total obligations (MVP - no recurring yet)
	closingBalance := startingBalance.Sub(totalObligations)

	// Build month boundaries
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local) // Last day of month

	// For future months, days remaining = total days in month
	daysRemaining := endDate.Day()

	// Disposable = starting - total obligations (loan payments + deferred CC)
	disposableIncome := startingBalance.Sub(totalObligations)

	// Calculate daily budget based on disposable income
	dailyBudget := decimal.Zero
	if daysRemaining > 0 {
		dailyBudget = disposableIncome.Div(decimal.NewFromInt(int64(daysRemaining)))
	}

	return &domain.DashboardSummary{
		IsProjection:          true,
		ProjectionLimitMonths: domain.MaxProjectionMonths,
		TotalBalance:          startingBalance,
		InHandBalance:         startingBalance,
		DisposableIncome:      disposableIncome,
		UnpaidExpenses:        deferredFromPrevMonth, // Show deferred CC as unpaid expenses in projection
		UnpaidLoanPayments:    unpaidLoanPayments,
		DaysRemaining:         daysRemaining,
		DailyBudget:           dailyBudget,
		Month: &domain.CalculatedMonth{
			Month: domain.Month{
				Year:            year,
				Month:           month,
				StartDate:       startDate,
				EndDate:         endDate,
				StartingBalance: startingBalance,
			},
			TotalIncome:    decimal.Zero,
			TotalExpenses:  decimal.Zero,
			ClosingBalance: closingBalance,
		},
		Projection: projection,
	}, nil
}

// chainBalanceToMonth calculates the projected starting balance for a future month
// For MVP, uses the current month's closing balance as the projection base
func (s *DashboardService) chainBalanceToMonth(workspaceID int32) (decimal.Decimal, error) {
	now := time.Now()
	currentMonth, err := s.monthService.GetOrCreateMonth(workspaceID, now.Year(), int(now.Month()))
	if err != nil {
		return decimal.Zero, err
	}

	return currentMonth.ClosingBalance, nil
}

// monthsBetween calculates the number of months between two dates
func monthsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months
}

// calculateDaysRemaining returns the number of days from today until the end of the month
// If viewing a past month, returns 0. If viewing a future month, returns days in that month.
func (s *DashboardService) calculateDaysRemaining(year, month int) int {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// End of the target month (last day at midnight)
	endOfMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local)

	// Start of the target month
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

	// If viewing a past month, return 0
	if endOfMonth.Before(today) {
		return 0
	}

	// If viewing a future month, return total days in that month
	if startOfMonth.After(today) {
		return endOfMonth.Day()
	}

	// Current month: calculate days from today to end of month (inclusive)
	daysRemaining := int(endOfMonth.Sub(today).Hours()/24) + 1
	if daysRemaining < 1 {
		daysRemaining = 1
	}

	return daysRemaining
}

// calculateTotalBalance calculates total balance from all accounts
// Total = sum of all calculated balances
// Note: For liabilities (CC), the calculated balance is negative when debt exists,
// so we simply sum all balances to get net worth
func (s *DashboardService) calculateTotalBalance(workspaceID int32) (decimal.Decimal, error) {
	balances, err := s.calcService.CalculateAccountBalances(workspaceID)
	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, balance := range balances {
		total = total.Add(balance.CalculatedBalance)
	}

	return total, nil
}

// GetFutureSpending returns aggregated spending data for future months
// including both actual and projected transactions
func (s *DashboardService) GetFutureSpending(workspaceID int32, months int) (*domain.FutureSpendingData, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, months, 0)

	// Get all transactions in the date range (includes both actual and projected)
	// Uses dedicated aggregation method that doesn't have pagination limits
	transactions, err := s.transactionRepo.GetByDateRangeForAggregation(workspaceID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get accounts for name lookup
	accounts, err := s.accountRepo.GetAllByWorkspace(workspaceID, false)
	if err != nil {
		return nil, err
	}

	accountMap := make(map[int32]string)
	for _, acc := range accounts {
		accountMap[acc.ID] = acc.Name
	}

	// Get deferred CC transactions that should be carried forward
	deferredCC, err := s.transactionRepo.GetDeferredForSettlement(workspaceID)
	if err != nil {
		return nil, err
	}

	// Aggregate by month
	monthlyData := make(map[string]*domain.MonthSpending)

	// Process regular transactions
	for _, txn := range transactions {
		// Only count expenses
		if txn.Type != domain.TransactionTypeExpense {
			continue
		}

		monthKey := txn.TransactionDate.Format("2006-01")

		if _, exists := monthlyData[monthKey]; !exists {
			monthlyData[monthKey] = &domain.MonthSpending{
				Month:      monthKey,
				Total:      decimal.Zero,
				ByCategory: make(map[int32]decimal.Decimal),
				ByAccount:  make(map[int32]decimal.Decimal),
			}
		}

		m := monthlyData[monthKey]
		amount := txn.Amount.Abs()
		m.Total = m.Total.Add(amount)
		if txn.CategoryID != nil {
			m.ByCategory[*txn.CategoryID] = m.ByCategory[*txn.CategoryID].Add(amount)
		}
		m.ByAccount[txn.AccountID] = m.ByAccount[txn.AccountID].Add(amount)
	}

	// Add deferred CC to the current month (they're carried forward)
	currentMonthKey := startDate.Format("2006-01")
	if _, exists := monthlyData[currentMonthKey]; !exists {
		monthlyData[currentMonthKey] = &domain.MonthSpending{
			Month:      currentMonthKey,
			Total:      decimal.Zero,
			ByCategory: make(map[int32]decimal.Decimal),
			ByAccount:  make(map[int32]decimal.Decimal),
		}
	}
	for _, txn := range deferredCC {
		m := monthlyData[currentMonthKey]
		amount := txn.Amount.Abs()
		m.Total = m.Total.Add(amount)
		if txn.CategoryID != nil {
			m.ByCategory[*txn.CategoryID] = m.ByCategory[*txn.CategoryID].Add(amount)
		}
		m.ByAccount[txn.AccountID] = m.ByAccount[txn.AccountID].Add(amount)
	}

	// Build category name map (from transactions that have category names)
	categoryMap := make(map[int32]string)
	for _, txn := range transactions {
		if txn.CategoryID != nil && txn.CategoryName != nil {
			categoryMap[*txn.CategoryID] = *txn.CategoryName
		}
	}

	// Convert to response format, ensuring all months in range are present
	result := &domain.FutureSpendingData{
		Months: make([]domain.MonthSpendingResponse, 0, months),
	}

	current := startDate
	for current.Before(endDate) {
		monthKey := current.Format("2006-01")

		data, exists := monthlyData[monthKey]
		if !exists {
			data = &domain.MonthSpending{
				Month:      monthKey,
				Total:      decimal.Zero,
				ByCategory: make(map[int32]decimal.Decimal),
				ByAccount:  make(map[int32]decimal.Decimal),
			}
		}

		response := domain.MonthSpendingResponse{
			Month:      monthKey,
			Total:      data.Total.StringFixed(2),
			ByCategory: make([]domain.CategoryAmount, 0, len(data.ByCategory)),
			ByAccount:  make([]domain.AccountAmount, 0, len(data.ByAccount)),
		}

		for catID, amount := range data.ByCategory {
			response.ByCategory = append(response.ByCategory, domain.CategoryAmount{
				ID:     catID,
				Name:   categoryMap[catID],
				Amount: amount.StringFixed(2),
			})
		}

		for accID, amount := range data.ByAccount {
			response.ByAccount = append(response.ByAccount, domain.AccountAmount{
				ID:     accID,
				Name:   accountMap[accID],
				Amount: amount.StringFixed(2),
			})
		}

		result.Months = append(result.Months, response)
		current = current.AddDate(0, 1, 0)
	}

	return result, nil
}
