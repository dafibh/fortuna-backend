package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// LoanPaymentService handles loan payment business logic
type LoanPaymentService struct {
	pool           *pgxpool.Pool
	paymentRepo    domain.LoanPaymentRepository
	loanRepo       domain.LoanRepository
	providerRepo   domain.LoanProviderRepository
	eventPublisher websocket.EventPublisher
}

// NewLoanPaymentService creates a new LoanPaymentService
func NewLoanPaymentService(pool *pgxpool.Pool, paymentRepo domain.LoanPaymentRepository, loanRepo domain.LoanRepository, providerRepo domain.LoanProviderRepository) *LoanPaymentService {
	return &LoanPaymentService{
		pool:         pool,
		paymentRepo:  paymentRepo,
		loanRepo:     loanRepo,
		providerRepo: providerRepo,
	}
}

// SetEventPublisher sets the event publisher for real-time updates
func (s *LoanPaymentService) SetEventPublisher(publisher websocket.EventPublisher) {
	s.eventPublisher = publisher
}

// publishEvent publishes a WebSocket event if a publisher is configured
func (s *LoanPaymentService) publishEvent(workspaceID int32, event websocket.Event) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(workspaceID, event)
	}
}

// GetPaymentsByLoanID retrieves all payments for a loan, validating workspace ownership
func (s *LoanPaymentService) GetPaymentsByLoanID(workspaceID int32, loanID int32) ([]*domain.LoanPayment, error) {
	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	return s.paymentRepo.GetByLoanID(loanID)
}

// UpdatePaymentAmount updates the amount for a specific payment
func (s *LoanPaymentService) UpdatePaymentAmount(workspaceID int32, loanID int32, paymentID int32, amount decimal.Decimal) (*domain.LoanPayment, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrLoanPaymentAmountInvalid
	}

	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	// Verify payment belongs to loan
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		return nil, err
	}
	if payment.LoanID != loanID {
		return nil, domain.ErrLoanPaymentNotFound
	}

	return s.paymentRepo.UpdateAmount(paymentID, amount)
}

// TogglePaymentPaid toggles the paid status of a payment
// If customPaidDate is provided and paid is true, uses that date; otherwise defaults to current date
func (s *LoanPaymentService) TogglePaymentPaid(workspaceID int32, loanID int32, paymentID int32, paid bool, customPaidDate *time.Time) (*domain.LoanPayment, error) {
	// Verify loan belongs to workspace
	_, err := s.loanRepo.GetByID(workspaceID, loanID)
	if err != nil {
		return nil, err
	}

	// Verify payment belongs to loan
	payment, err := s.paymentRepo.GetByID(paymentID)
	if err != nil {
		return nil, err
	}
	if payment.LoanID != loanID {
		return nil, domain.ErrLoanPaymentNotFound
	}

	var paidDate *time.Time
	if paid {
		if customPaidDate != nil {
			paidDate = customPaidDate
		} else {
			now := time.Now()
			paidDate = &now
		}
	}

	return s.paymentRepo.TogglePaid(paymentID, paid, paidDate)
}

// GetPaymentsByMonth retrieves all loan payments due in a specific month for a workspace
func (s *LoanPaymentService) GetPaymentsByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return s.paymentRepo.GetByMonth(workspaceID, year, month)
}

// GetUnpaidPaymentsByMonth retrieves unpaid loan payments due in a specific month
func (s *LoanPaymentService) GetUnpaidPaymentsByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return s.paymentRepo.GetUnpaidByMonth(workspaceID, year, month)
}

// GetEarliestUnpaidMonth retrieves the earliest unpaid month for a provider.
// Returns nil if there are no unpaid months (all payments are complete).
func (s *LoanPaymentService) GetEarliestUnpaidMonth(workspaceID int32, providerID int32) (*domain.EarliestUnpaidMonth, error) {
	// Validate provider exists and belongs to workspace
	_, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		return nil, err
	}

	return s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
}

// PayMonth atomically marks all loan payments for a specific provider-month as paid.
// Validates sequential enforcement: payments must be made in order (earliest unpaid month first).
// Only works for providers with payment_mode = 'consolidated_monthly'.
func (s *LoanPaymentService) PayMonth(ctx context.Context, workspaceID int32, providerID int32, month string, paymentIDs []int32) (*domain.PayMonthResult, error) {
	// 1. Validate provider exists and belongs to workspace
	provider, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		return nil, err
	}

	// 2. Validate provider uses consolidated_monthly mode
	if provider.PaymentMode != domain.PaymentModeConsolidatedMonthly {
		return nil, domain.ErrProviderNotConsolidated
	}

	// 3. Parse target month
	targetYear, targetMonth, err := parseMonth(month)
	if err != nil {
		return nil, err
	}

	// 4. Validate sequential enforcement (target month = earliest unpaid)
	earliestUnpaid, err := s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
	if err != nil {
		return nil, err
	}
	if earliestUnpaid == nil {
		return nil, domain.ErrNoUnpaidMonths
	}

	if targetYear != int(earliestUnpaid.Year) || targetMonth != int(earliestUnpaid.Month) {
		return nil, domain.ErrMustPayEarlierMonth{
			Expected:  fmt.Sprintf("%04d-%02d", earliestUnpaid.Year, earliestUnpaid.Month),
			Requested: month,
		}
	}

	// 5. Validate all payment IDs belong to that month
	expectedPayments, err := s.paymentRepo.GetUnpaidPaymentsByProviderMonth(workspaceID, providerID, int32(targetYear), int32(targetMonth))
	if err != nil {
		return nil, err
	}

	if len(paymentIDs) == 0 {
		return nil, domain.ErrPaymentIDsInvalid
	}

	// Build map of expected payment IDs
	expectedIDMap := make(map[int32]bool)
	for _, p := range expectedPayments {
		expectedIDMap[p.ID] = true
	}

	// Verify all provided payment IDs are in the expected list
	for _, id := range paymentIDs {
		if !expectedIDMap[id] {
			return nil, domain.ErrPaymentIDsInvalid
		}
	}

	// 6. Begin transaction and batch update
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	paidCount, totalAmount, err := s.paymentRepo.BatchUpdatePaidTx(tx, paymentIDs, workspaceID)
	if err != nil {
		return nil, err
	}

	// 7. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 8. Get next payable month
	var nextPayableMonth *string
	nextUnpaid, err := s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
	if err == nil && nextUnpaid != nil {
		nextMonth := fmt.Sprintf("%04d-%02d", nextUnpaid.Year, nextUnpaid.Month)
		nextPayableMonth = &nextMonth
	}

	now := time.Now()
	return &domain.PayMonthResult{
		Month:            month,
		PaidCount:        paidCount,
		TotalAmount:      totalAmount,
		PaidAt:           now,
		NextPayableMonth: nextPayableMonth,
	}, nil
}

// ValidatePayMonth validates whether a month can be paid for a provider
// without actually performing the payment. Used for pre-validation.
func (s *LoanPaymentService) ValidatePayMonth(ctx context.Context, workspaceID int32, providerID int32, month string, paymentIDs []int32) error {
	// 1. Validate provider exists and belongs to workspace
	provider, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		return err
	}

	// 2. Validate provider uses consolidated_monthly mode
	if provider.PaymentMode != domain.PaymentModeConsolidatedMonthly {
		return domain.ErrProviderNotConsolidated
	}

	// 3. Parse target month
	targetYear, targetMonth, err := parseMonth(month)
	if err != nil {
		return err
	}

	// 4. Validate sequential enforcement
	earliestUnpaid, err := s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
	if err != nil {
		return err
	}
	if earliestUnpaid == nil {
		return domain.ErrNoUnpaidMonths
	}

	if targetYear != int(earliestUnpaid.Year) || targetMonth != int(earliestUnpaid.Month) {
		return domain.ErrMustPayEarlierMonth{
			Expected:  fmt.Sprintf("%04d-%02d", earliestUnpaid.Year, earliestUnpaid.Month),
			Requested: month,
		}
	}

	// 5. Validate payment IDs
	expectedPayments, err := s.paymentRepo.GetUnpaidPaymentsByProviderMonth(workspaceID, providerID, int32(targetYear), int32(targetMonth))
	if err != nil {
		return err
	}

	if len(paymentIDs) == 0 {
		return domain.ErrPaymentIDsInvalid
	}

	expectedIDMap := make(map[int32]bool)
	for _, p := range expectedPayments {
		expectedIDMap[p.ID] = true
	}

	for _, id := range paymentIDs {
		if !expectedIDMap[id] {
			return domain.ErrPaymentIDsInvalid
		}
	}

	return nil
}

// parseMonth parses a "YYYY-MM" formatted string into year and month integers
func parseMonth(month string) (year, monthNum int, err error) {
	// Validate format: must be exactly 7 characters (YYYY-MM)
	if len(month) != 7 || month[4] != '-' {
		return 0, 0, fmt.Errorf("invalid month format, expected YYYY-MM")
	}

	_, err = fmt.Sscanf(month, "%04d-%02d", &year, &monthNum)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month format, expected YYYY-MM: %w", err)
	}
	if monthNum < 1 || monthNum > 12 {
		return 0, 0, fmt.Errorf("month must be between 1 and 12")
	}
	return year, monthNum, nil
}

// formatMonth formats year and month integers into "YYYY-MM" string
func formatMonth(year, month int) string {
	return fmt.Sprintf("%04d-%02d", year, month)
}

// nextMonth returns the next month after the given year/month
func nextMonth(year, month int) (int, int) {
	if month == 12 {
		return year + 1, 1
	}
	return year, month + 1
}

// compareMonths compares two months. Returns -1 if a < b, 0 if a == b, 1 if a > b
func compareMonths(yearA, monthA, yearB, monthB int) int {
	if yearA < yearB {
		return -1
	}
	if yearA > yearB {
		return 1
	}
	// Same year
	if monthA < monthB {
		return -1
	}
	if monthA > monthB {
		return 1
	}
	return 0
}

// generateMonthRange generates a list of months from start to end (inclusive)
func generateMonthRange(startYear, startMonth, endYear, endMonth int) []string {
	var months []string
	year, month := startYear, startMonth
	for {
		months = append(months, formatMonth(year, month))
		if year == endYear && month == endMonth {
			break
		}
		year, month = nextMonth(year, month)
	}
	return months
}

// PayRange atomically marks all loan payments for a range of consecutive months as paid.
// Validates sequential enforcement: start month must be the earliest unpaid month.
// Validates consecutive months: all months from start to end must be present.
// Only works for providers with payment_mode = 'consolidated_monthly'.
func (s *LoanPaymentService) PayRange(ctx context.Context, workspaceID int32, providerID int32, startMonth string, endMonth string, paymentIDs []int32) (*domain.PayRangeResult, error) {
	// 1. Validate provider exists and belongs to workspace
	provider, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		return nil, err
	}

	// 2. Validate provider uses consolidated_monthly mode
	if provider.PaymentMode != domain.PaymentModeConsolidatedMonthly {
		return nil, domain.ErrProviderNotConsolidated
	}

	// 3. Parse start and end months
	startYear, startMonthNum, err := parseMonth(startMonth)
	if err != nil {
		return nil, err
	}

	endYear, endMonthNum, err := parseMonth(endMonth)
	if err != nil {
		return nil, err
	}

	// 4. Validate end month is after start month
	if compareMonths(endYear, endMonthNum, startYear, startMonthNum) <= 0 {
		return nil, domain.ErrEndMonthBeforeStart
	}

	// 5. Validate sequential enforcement (start month = earliest unpaid)
	earliestUnpaid, err := s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
	if err != nil {
		return nil, err
	}
	if earliestUnpaid == nil {
		return nil, domain.ErrNoUnpaidMonths
	}

	if startYear != int(earliestUnpaid.Year) || startMonthNum != int(earliestUnpaid.Month) {
		return nil, domain.ErrMustPayEarlierMonth{
			Expected:  formatMonth(int(earliestUnpaid.Year), int(earliestUnpaid.Month)),
			Requested: startMonth,
		}
	}

	// 6. Generate expected month range
	expectedMonths := generateMonthRange(startYear, startMonthNum, endYear, endMonthNum)

	// 7. Validate payment IDs and check they cover all months in range
	if len(paymentIDs) == 0 {
		return nil, domain.ErrPaymentIDsInvalid
	}

	// Collect all unpaid payments for the entire range and validate
	allExpectedPayments := make([]*domain.LoanPayment, 0)
	for _, month := range expectedMonths {
		year, monthNum, _ := parseMonth(month)
		payments, err := s.paymentRepo.GetUnpaidPaymentsByProviderMonth(workspaceID, providerID, int32(year), int32(monthNum))
		if err != nil {
			return nil, err
		}
		if len(payments) == 0 {
			// No payments for this month - gap detected
			return nil, domain.ErrCannotSkipMonth{Skipped: month}
		}
		allExpectedPayments = append(allExpectedPayments, payments...)
	}

	// Build map of expected payment IDs
	expectedIDMap := make(map[int32]bool)
	for _, p := range allExpectedPayments {
		expectedIDMap[p.ID] = true
	}

	// Verify all provided payment IDs are in the expected list
	for _, id := range paymentIDs {
		if !expectedIDMap[id] {
			return nil, domain.ErrPaymentIDsInvalid
		}
	}

	// 8. Begin transaction and batch update
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	paidCount, totalAmount, err := s.paymentRepo.BatchUpdatePaidTx(tx, paymentIDs, workspaceID)
	if err != nil {
		return nil, err
	}

	// 9. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 10. Get next payable month
	var nextPayableMonth *string
	nextUnpaid, err := s.paymentRepo.GetEarliestUnpaidMonth(workspaceID, providerID)
	if err == nil && nextUnpaid != nil {
		next := formatMonth(int(nextUnpaid.Year), int(nextUnpaid.Month))
		nextPayableMonth = &next
	}

	now := time.Now()
	result := &domain.PayRangeResult{
		MonthsPaid:       expectedMonths,
		PaidCount:        paidCount,
		TotalAmount:      totalAmount,
		PaidAt:           now,
		NextPayableMonth: nextPayableMonth,
	}

	// 11. Publish WebSocket event
	eventPayload := map[string]interface{}{
		"providerId":       providerID,
		"monthsPaid":       result.MonthsPaid,
		"paidCount":        result.PaidCount,
		"totalAmount":      result.TotalAmount.StringFixed(2),
		"paidAt":           result.PaidAt.Format(time.RFC3339),
		"nextPayableMonth": result.NextPayableMonth,
	}
	s.publishEvent(workspaceID, websocket.LoanPaymentBatchPaid(eventPayload))

	return result, nil
}

// UnpayMonth atomically marks all loan payments for a specific month as unpaid.
// Validates reverse sequential enforcement: can only unpay the latest paid month.
// Only works for providers with payment_mode = 'consolidated_monthly'.
func (s *LoanPaymentService) UnpayMonth(ctx context.Context, workspaceID int32, providerID int32, month string) (*domain.UnpayMonthResult, error) {
	// 1. Validate provider exists and belongs to workspace
	provider, err := s.providerRepo.GetByID(workspaceID, providerID)
	if err != nil {
		return nil, err
	}

	// 2. Validate provider uses consolidated_monthly mode
	if provider.PaymentMode != domain.PaymentModeConsolidatedMonthly {
		return nil, domain.ErrProviderNotConsolidated
	}

	// 3. Parse target month
	targetYear, targetMonth, err := parseMonth(month)
	if err != nil {
		return nil, err
	}

	// 4. Validate reverse sequential enforcement (can only unpay latest paid month)
	latestPaid, err := s.paymentRepo.GetLatestPaidMonth(workspaceID, providerID)
	if err != nil {
		return nil, err
	}
	if latestPaid == nil {
		return nil, domain.ErrNoPaidMonths
	}

	// Check if target month is the latest paid month
	if targetYear != int(latestPaid.Year) || targetMonth != int(latestPaid.Month) {
		return nil, domain.ErrCannotUnpayEarlierMonth{
			Latest:    fmt.Sprintf("%04d-%02d", latestPaid.Year, latestPaid.Month),
			Requested: month,
		}
	}

	// 5. Get all paid payments for this month
	paidPayments, err := s.paymentRepo.GetPaidPaymentsByProviderMonth(workspaceID, providerID, int32(targetYear), int32(targetMonth))
	if err != nil {
		return nil, err
	}
	if len(paidPayments) == 0 {
		return nil, domain.ErrNoPaidMonths
	}

	// Collect payment IDs and calculate total amount
	paymentIDs := make([]int32, len(paidPayments))
	totalAmount := decimal.Zero
	for i, p := range paidPayments {
		paymentIDs[i] = p.ID
		totalAmount = totalAmount.Add(p.Amount)
	}

	// 6. Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 7. Batch update payments to unpaid
	unpaidCount, err := s.paymentRepo.BatchUpdateUnpaidTx(tx, paymentIDs)
	if err != nil {
		return nil, err
	}

	// 8. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 9. Build result - the unpaid month is now the previous payable month
	result := &domain.UnpayMonthResult{
		Month:           month,
		UnpaidCount:     unpaidCount,
		TotalAmount:     totalAmount,
		PreviousPayable: &month,
	}

	// 10. Publish WebSocket event
	unpaEventPayload := map[string]interface{}{
		"providerId":  providerID,
		"month":       result.Month,
		"unpaidCount": result.UnpaidCount,
		"totalAmount": result.TotalAmount.StringFixed(2),
	}
	s.publishEvent(workspaceID, websocket.LoanPaymentBatchUnpaid(unpaEventPayload))

	return result, nil
}
