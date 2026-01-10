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

	// 3. Calculate in-hand balance (starting + paid income - paid expenses only)
	paidIncome, err := s.transactionRepo.SumPaidIncomeByDateRange(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	paidExpenses, err := s.transactionRepo.SumPaidExpensesByDateRange(
		workspaceID, monthData.StartDate, monthData.EndDate)
	if err != nil {
		return nil, err
	}

	inHandBalance := monthData.StartingBalance.Add(paidIncome).Sub(paidExpenses)

	// 4. Calculate unpaid expenses for disposable income
	unpaidExpenses, err := s.transactionRepo.SumUnpaidExpensesByDateRange(
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

	// 8. Get CC payable summary
	ccPayable, err := s.GetCCPayable(workspaceID)
	if err != nil {
		return nil, err
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
		CCPayable:          ccPayable,
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

	// Build projection details with loan payments
	projection := &domain.ProjectionDetails{
		RecurringIncome:   decimal.Zero,
		RecurringExpenses: decimal.Zero,
		LoanPayments:      unpaidLoanPayments,
		Note:              "Recurring transactions not yet configured",
	}

	// Projected closing = starting - loan payments (MVP - no recurring yet)
	closingBalance := startingBalance.Sub(unpaidLoanPayments)

	// Build month boundaries
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local) // Last day of month

	// For future months, days remaining = total days in month
	daysRemaining := endDate.Day()

	// Disposable = starting - loan payments (no unpaid expenses in future)
	disposableIncome := startingBalance.Sub(unpaidLoanPayments)

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
		UnpaidExpenses:        decimal.Zero,
		UnpaidLoanPayments:    unpaidLoanPayments,
		DaysRemaining:         daysRemaining,
		DailyBudget:           dailyBudget,
		CCPayable: &domain.CCPayableSummary{
			ThisMonth: decimal.Zero,
			NextMonth: decimal.Zero,
			Total:     decimal.Zero,
		},
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

// GetCCPayable calculates the CC payable summary for a workspace
func (s *DashboardService) GetCCPayable(workspaceID int32) (*domain.CCPayableSummary, error) {
	rows, err := s.transactionRepo.GetCCPayableSummary(workspaceID)
	if err != nil {
		return nil, err
	}

	summary := &domain.CCPayableSummary{
		ThisMonth: decimal.Zero,
		NextMonth: decimal.Zero,
	}

	for _, row := range rows {
		switch row.SettlementIntent {
		case domain.CCSettlementThisMonth:
			summary.ThisMonth = row.Total
		case domain.CCSettlementNextMonth:
			summary.NextMonth = row.Total
		}
	}

	summary.Total = summary.ThisMonth.Add(summary.NextMonth)
	return summary, nil
}

// GetFutureSpending returns aggregated spending data for future months
func (s *DashboardService) GetFutureSpending(workspaceID int32, months int) (*domain.FutureSpending, error) {
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	result := &domain.FutureSpending{
		Months: make([]domain.FutureSpendingMonth, months),
	}

	// Get all expense transactions for the date range (current month + future months)
	endDate := currentMonth.AddDate(0, months, 0)
	startDate := currentMonth

	// Get all expense transactions in the date range
	transactions, err := s.transactionRepo.GetExpensesByDateRange(workspaceID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get account names for mapping
	accounts, err := s.accountRepo.GetAllByWorkspace(workspaceID, false)
	if err != nil {
		return nil, err
	}
	accountNames := make(map[int32]string)
	for _, acc := range accounts {
		accountNames[acc.ID] = acc.Name
	}

	// Get category names for mapping
	categoryNames, err := s.getCategoryNames(workspaceID)
	if err != nil {
		return nil, err
	}

	// Group transactions by month
	monthData := make(map[string]*monthAggregation)

	// Initialize all months
	for i := 0; i < months; i++ {
		monthDate := currentMonth.AddDate(0, i, 0)
		monthKey := monthDate.Format("2006-01")
		monthData[monthKey] = &monthAggregation{
			total:      decimal.Zero,
			byCategory: make(map[int32]decimal.Decimal),
			byAccount:  make(map[int32]decimal.Decimal),
		}
	}

	// Aggregate transactions
	for _, tx := range transactions {
		monthKey := tx.TransactionDate.Format("2006-01")
		if agg, ok := monthData[monthKey]; ok {
			agg.total = agg.total.Add(tx.Amount)

			// By category
			catID := int32(0)
			if tx.CategoryID != nil {
				catID = *tx.CategoryID
			}
			agg.byCategory[catID] = agg.byCategory[catID].Add(tx.Amount)

			// By account
			agg.byAccount[tx.AccountID] = agg.byAccount[tx.AccountID].Add(tx.Amount)
		}
	}

	// Build result
	for i := 0; i < months; i++ {
		monthDate := currentMonth.AddDate(0, i, 0)
		monthKey := monthDate.Format("2006-01")
		agg := monthData[monthKey]

		// Convert category map to slice
		byCategory := make([]domain.FutureSpendingCategory, 0, len(agg.byCategory))
		for catID, amount := range agg.byCategory {
			catName := "Uncategorized"
			if catID > 0 {
				if name, ok := categoryNames[catID]; ok {
					catName = name
				}
			}
			byCategory = append(byCategory, domain.FutureSpendingCategory{
				ID:     catID,
				Name:   catName,
				Amount: amount,
			})
		}

		// Convert account map to slice
		byAccount := make([]domain.FutureSpendingAccount, 0, len(agg.byAccount))
		for accID, amount := range agg.byAccount {
			accName := "Unknown"
			if name, ok := accountNames[accID]; ok {
				accName = name
			}
			byAccount = append(byAccount, domain.FutureSpendingAccount{
				ID:     accID,
				Name:   accName,
				Amount: amount,
			})
		}

		result.Months[i] = domain.FutureSpendingMonth{
			Month:      monthKey,
			Total:      agg.total,
			ByCategory: byCategory,
			ByAccount:  byAccount,
		}
	}

	return result, nil
}

// monthAggregation holds aggregated data for a single month
type monthAggregation struct {
	total      decimal.Decimal
	byCategory map[int32]decimal.Decimal
	byAccount  map[int32]decimal.Decimal
}

// getCategoryNames returns a map of category ID to name
func (s *DashboardService) getCategoryNames(workspaceID int32) (map[int32]string, error) {
	// This is a simple placeholder - in production you'd have a category repository
	// For now, return empty map and let the handler handle unknown categories
	return make(map[int32]string), nil
}
