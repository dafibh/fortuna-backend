package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// MonthService handles month-related business logic
type MonthService struct {
	monthRepo       domain.MonthRepository
	transactionRepo domain.TransactionRepository
	calcService     *CalculationService
}

// NewMonthService creates a new MonthService
func NewMonthService(monthRepo domain.MonthRepository, transactionRepo domain.TransactionRepository, calcService *CalculationService) *MonthService {
	return &MonthService{
		monthRepo:       monthRepo,
		transactionRepo: transactionRepo,
		calcService:     calcService,
	}
}

// GetOrCreateMonth ensures a month record exists, creating if needed
func (s *MonthService) GetOrCreateMonth(workspaceID int32, year, month int) (*domain.CalculatedMonth, error) {
	// Validate month
	if month < 1 || month > 12 {
		return nil, domain.ErrInvalidInput
	}
	if year < 2000 || year > 2100 {
		return nil, domain.ErrInvalidInput
	}

	// Try to get existing month
	m, err := s.monthRepo.GetByYearMonth(workspaceID, year, month)
	if err != nil && !errors.Is(err, domain.ErrMonthNotFound) {
		return nil, err
	}

	// If not exists, calculate starting balance and create
	if errors.Is(err, domain.ErrMonthNotFound) {
		startingBalance, err := s.calculateStartingBalance(workspaceID, year, month)
		if err != nil {
			return nil, err
		}

		startDate, endDate := getMonthBoundaries(year, month)
		m, err = s.monthRepo.Create(workspaceID, year, month, startDate, endDate, startingBalance)
		if err != nil {
			// Handle race condition: if another request created the month concurrently,
			// retry the read instead of failing
			m, retryErr := s.monthRepo.GetByYearMonth(workspaceID, year, month)
			if retryErr != nil {
				// Return the original create error if retry also failed
				return nil, err
			}
			// Successfully retrieved the month created by concurrent request
			return s.enrichWithCalculations(m)
		}
	}

	// Calculate income/expenses/closing balance from transactions
	return s.enrichWithCalculations(m)
}

// GetMonth retrieves a month without creating it
func (s *MonthService) GetMonth(workspaceID int32, year, month int) (*domain.CalculatedMonth, error) {
	m, err := s.monthRepo.GetByYearMonth(workspaceID, year, month)
	if err != nil {
		return nil, err
	}
	return s.enrichWithCalculations(m)
}

// GetAllMonths retrieves all months for a workspace with calculations (optimized batch query)
func (s *MonthService) GetAllMonths(workspaceID int32) ([]*domain.CalculatedMonth, error) {
	months, err := s.monthRepo.GetAll(workspaceID)
	if err != nil {
		return nil, err
	}

	if len(months) == 0 {
		return []*domain.CalculatedMonth{}, nil
	}

	// Batch fetch all monthly summaries in a single query (N+1 prevention)
	summaries, err := s.transactionRepo.GetMonthlyTransactionSummaries(workspaceID)
	if err != nil {
		return nil, err
	}

	// Build lookup map: "year-month" -> summary
	summaryMap := make(map[string]*domain.MonthlyTransactionSummary)
	for _, s := range summaries {
		key := monthSummaryKey(s.Year, s.Month)
		summaryMap[key] = s
	}

	// Enrich months with pre-fetched summaries
	result := make([]*domain.CalculatedMonth, len(months))
	for i, m := range months {
		key := monthSummaryKey(m.Year, m.Month)
		summary := summaryMap[key]

		income := decimal.Zero
		expenses := decimal.Zero
		if summary != nil {
			income = summary.TotalIncome
			expenses = summary.TotalExpenses
		}

		result[i] = &domain.CalculatedMonth{
			Month:          *m,
			TotalIncome:    income,
			TotalExpenses:  expenses,
			ClosingBalance: m.StartingBalance.Add(income).Sub(expenses),
		}
	}
	return result, nil
}

// monthSummaryKey generates a lookup key for monthly summaries
func monthSummaryKey(year, month int) string {
	return fmt.Sprintf("%d-%d", year, month)
}

// calculateStartingBalance determines starting balance for a month
func (s *MonthService) calculateStartingBalance(workspaceID int32, year, month int) (decimal.Decimal, error) {
	// Check if previous month exists
	prevYear, prevMonth := getPreviousMonth(year, month)
	prevMonthData, err := s.monthRepo.GetByYearMonth(workspaceID, prevYear, prevMonth)

	if err == nil {
		// Previous month exists - use its closing balance
		closing, err := s.calculateClosingBalance(prevMonthData)
		if err != nil {
			return decimal.Zero, err
		}
		return closing, nil
	}

	if !errors.Is(err, domain.ErrMonthNotFound) {
		return decimal.Zero, err
	}

	// No previous month - calculate from current account balances
	return s.getTotalAccountBalance(workspaceID)
}

// getTotalAccountBalance calculates the total balance across all accounts
func (s *MonthService) getTotalAccountBalance(workspaceID int32) (decimal.Decimal, error) {
	balances, err := s.calcService.CalculateAccountBalances(workspaceID)
	if err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, b := range balances {
		total = total.Add(b.CalculatedBalance)
	}
	return total, nil
}

// calculateClosingBalance calculates closing balance for a month
func (s *MonthService) calculateClosingBalance(m *domain.Month) (decimal.Decimal, error) {
	startDate, endDate := getMonthBoundaries(m.Year, m.Month)

	income, err := s.transactionRepo.SumByTypeAndDateRange(m.WorkspaceID, startDate, endDate, domain.TransactionTypeIncome)
	if err != nil {
		return decimal.Zero, err
	}

	expenses, err := s.transactionRepo.SumByTypeAndDateRange(m.WorkspaceID, startDate, endDate, domain.TransactionTypeExpense)
	if err != nil {
		return decimal.Zero, err
	}

	return m.StartingBalance.Add(income).Sub(expenses), nil
}

// enrichWithCalculations adds calculated fields to month
func (s *MonthService) enrichWithCalculations(m *domain.Month) (*domain.CalculatedMonth, error) {
	startDate, endDate := getMonthBoundaries(m.Year, m.Month)

	income, err := s.transactionRepo.SumByTypeAndDateRange(m.WorkspaceID, startDate, endDate, domain.TransactionTypeIncome)
	if err != nil {
		return nil, err
	}

	expenses, err := s.transactionRepo.SumByTypeAndDateRange(m.WorkspaceID, startDate, endDate, domain.TransactionTypeExpense)
	if err != nil {
		return nil, err
	}

	return &domain.CalculatedMonth{
		Month:          *m,
		TotalIncome:    income,
		TotalExpenses:  expenses,
		ClosingBalance: m.StartingBalance.Add(income).Sub(expenses),
	}, nil
}

// getPreviousMonth returns the year and month of the previous month
func getPreviousMonth(year, month int) (int, int) {
	if month == 1 {
		return year - 1, 12
	}
	return year, month - 1
}

// getMonthBoundaries returns the start and end dates of a month
func getMonthBoundaries(year, month int) (time.Time, time.Time) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1) // Last day of month
	return startDate, endDate
}
