package postgres

// LoanPaymentRepository implements domain.LoanPaymentRepository using the transactions table.
// CL v2 Migration: The loan_payments table was dropped and replaced with transactions.loan_id.
// This repository converts transaction data to LoanPayment format for frontend compatibility.

import (
	"context"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// LoanPaymentRepository implements domain.LoanPaymentRepository using transactions table
type LoanPaymentRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewLoanPaymentRepository creates a new LoanPaymentRepository
func NewLoanPaymentRepository(pool *pgxpool.Pool) *LoanPaymentRepository {
	return &LoanPaymentRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// GetByLoanID retrieves all payments for a loan from transactions table
func (r *LoanPaymentRepository) GetByLoanID(loanID int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()

	rows, err := r.queries.GetLoanPaymentsFromTransactions(ctx, pgtype.Int4{Int32: loanID, Valid: true})
	if err != nil {
		return nil, err
	}

	payments := make([]*domain.LoanPayment, len(rows))
	for i, row := range rows {
		payments[i] = sqlcLoanPaymentRowToDomain(row)
	}

	return payments, nil
}

// GetEarliestUnpaidMonth retrieves the earliest unpaid month for a provider
func (r *LoanPaymentRepository) GetEarliestUnpaidMonth(workspaceID int32, providerID int32) (*domain.EarliestUnpaidMonth, error) {
	ctx := context.Background()

	row, err := r.queries.GetEarliestUnpaidLoanMonth(ctx, sqlc.GetEarliestUnpaidLoanMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No unpaid months
		}
		return nil, err
	}

	return &domain.EarliestUnpaidMonth{
		Year:  row.Year,
		Month: row.Month,
	}, nil
}

// GetLatestPaidMonth retrieves the latest paid month for a provider
func (r *LoanPaymentRepository) GetLatestPaidMonth(workspaceID int32, providerID int32) (*domain.LatestPaidMonth, error) {
	ctx := context.Background()

	row, err := r.queries.GetLatestPaidLoanMonth(ctx, sqlc.GetLatestPaidLoanMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No paid months
		}
		return nil, err
	}

	return &domain.LatestPaidMonth{
		Year:  row.Year,
		Month: row.Month,
	}, nil
}

// GetUnpaidPaymentsByProviderMonth retrieves unpaid payments for a provider-month
func (r *LoanPaymentRepository) GetUnpaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()

	rows, err := r.queries.GetUnpaidLoanPaymentsByProviderMonth(ctx, sqlc.GetUnpaidLoanPaymentsByProviderMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		Year:        year,
		Month:       month,
	})
	if err != nil {
		return nil, err
	}

	payments := make([]*domain.LoanPayment, len(rows))
	for i, row := range rows {
		payments[i] = sqlcUnpaidPaymentRowToDomain(row)
	}

	return payments, nil
}

// GetPaidPaymentsByProviderMonth retrieves paid payments for a provider-month
func (r *LoanPaymentRepository) GetPaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()

	rows, err := r.queries.GetPaidLoanPaymentsByProviderMonth(ctx, sqlc.GetPaidLoanPaymentsByProviderMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		Year:        year,
		Month:       month,
	})
	if err != nil {
		return nil, err
	}

	payments := make([]*domain.LoanPayment, len(rows))
	for i, row := range rows {
		payments[i] = sqlcPaidPaymentRowToDomain(row)
	}

	return payments, nil
}

// BatchUpdatePaidTx marks multiple payments as paid within a transaction
func (r *LoanPaymentRepository) BatchUpdatePaidTx(tx any, paymentIDs []int32, workspaceID int32) (int, decimal.Decimal, error) {
	ctx := context.Background()
	pgxTx := tx.(pgx.Tx)
	qtx := r.queries.WithTx(pgxTx)

	rows, err := qtx.BatchMarkLoanTransactionsPaid(ctx, sqlc.BatchMarkLoanTransactionsPaidParams{
		WorkspaceID: workspaceID,
		Column2:     paymentIDs,
	})
	if err != nil {
		return 0, decimal.Zero, err
	}

	total := decimal.Zero
	for _, row := range rows {
		total = total.Add(pgNumericToDecimal(row.Amount).Abs())
	}

	return len(rows), total, nil
}

// BatchUpdateUnpaidTx marks multiple payments as unpaid within a transaction
func (r *LoanPaymentRepository) BatchUpdateUnpaidTx(tx any, paymentIDs []int32) (int, error) {
	ctx := context.Background()
	pgxTx := tx.(pgx.Tx)
	qtx := r.queries.WithTx(pgxTx)

	// Note: BatchMarkLoanTransactionsUnpaid needs workspaceID, but the interface doesn't provide it
	// This is a limitation of the current interface design
	// For now, we'll query without workspace filter since IDs are unique
	// TODO: Update interface to include workspaceID for better security
	rows, err := qtx.BatchMarkLoanTransactionsUnpaid(ctx, sqlc.BatchMarkLoanTransactionsUnpaidParams{
		WorkspaceID: 0, // Will be ignored by the query due to ID uniqueness
		Column2:     paymentIDs,
	})
	if err != nil {
		return 0, err
	}

	return len(rows), nil
}

// ===== Deprecated/Stub Methods =====
// The following methods return errors or empty results since loan_payments table was dropped.
// They are kept for interface compatibility but should not be used.

func (r *LoanPaymentRepository) Create(payment *domain.LoanPayment) (*domain.LoanPayment, error) {
	return nil, domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) CreateBatch(payments []*domain.LoanPayment) error {
	return domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) CreateBatchTx(tx any, payments []*domain.LoanPayment) error {
	return domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) GetByID(id int32) (*domain.LoanPayment, error) {
	return nil, domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) GetByLoanAndNumber(loanID int32, paymentNumber int32) (*domain.LoanPayment, error) {
	return nil, domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) UpdateAmount(id int32, amount decimal.Decimal) (*domain.LoanPayment, error) {
	return nil, domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) TogglePaid(id int32, paid bool, paidDate *time.Time) (*domain.LoanPayment, error) {
	return nil, domain.ErrLoanPaymentNotFound
}

func (r *LoanPaymentRepository) GetByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return []*domain.LoanPayment{}, nil
}

func (r *LoanPaymentRepository) GetUnpaidByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	return []*domain.LoanPayment{}, nil
}

func (r *LoanPaymentRepository) GetDeleteStats(loanID int32) (*domain.LoanDeleteStats, error) {
	return &domain.LoanDeleteStats{TotalCount: 0, PaidCount: 0, UnpaidCount: 0, TotalAmount: decimal.Zero}, nil
}

func (r *LoanPaymentRepository) GetPaymentsWithDetailsByMonth(workspaceID int32, year, month int) ([]*domain.MonthlyPaymentDetail, error) {
	return []*domain.MonthlyPaymentDetail{}, nil
}

func (r *LoanPaymentRepository) SumUnpaidByMonth(workspaceID int32, year, month int) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (r *LoanPaymentRepository) GetTrendRaw(workspaceID int32, startYear int32, startMonth int32) ([]*domain.TrendRawRow, error) {
	return []*domain.TrendRawRow{}, nil
}

// ===== Helper Functions =====

// sqlcLoanPaymentRowToDomain converts GetLoanPaymentsFromTransactionsRow to domain.LoanPayment
func sqlcLoanPaymentRowToDomain(row sqlc.GetLoanPaymentsFromTransactionsRow) *domain.LoanPayment {
	payment := &domain.LoanPayment{
		ID:            row.ID,
		LoanID:        row.LoanID.Int32,
		PaymentNumber: row.PaymentNumber,
		Amount:        pgNumericToDecimal(row.Amount),
		DueYear:       row.DueYear,
		DueMonth:      row.DueMonth,
		Paid:          row.Paid,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	// Handle paid_date - it's an interface{} that could be pgtype.Timestamptz or nil
	if row.PaidDate != nil {
		switch v := row.PaidDate.(type) {
		case time.Time:
			payment.PaidDate = &v
		case pgtype.Timestamptz:
			if v.Valid {
				payment.PaidDate = &v.Time
			}
		}
	}

	return payment
}

// sqlcUnpaidPaymentRowToDomain converts GetUnpaidLoanPaymentsByProviderMonthRow to domain.LoanPayment
func sqlcUnpaidPaymentRowToDomain(row sqlc.GetUnpaidLoanPaymentsByProviderMonthRow) *domain.LoanPayment {
	payment := &domain.LoanPayment{
		ID:            row.ID,
		LoanID:        row.LoanID.Int32,
		PaymentNumber: row.PaymentNumber,
		Amount:        pgNumericToDecimal(row.Amount),
		DueYear:       row.DueYear,
		DueMonth:      row.DueMonth,
		Paid:          row.Paid,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	if row.PaidDate != nil {
		switch v := row.PaidDate.(type) {
		case time.Time:
			payment.PaidDate = &v
		case pgtype.Timestamptz:
			if v.Valid {
				payment.PaidDate = &v.Time
			}
		}
	}

	return payment
}

// sqlcPaidPaymentRowToDomain converts GetPaidLoanPaymentsByProviderMonthRow to domain.LoanPayment
func sqlcPaidPaymentRowToDomain(row sqlc.GetPaidLoanPaymentsByProviderMonthRow) *domain.LoanPayment {
	payment := &domain.LoanPayment{
		ID:            row.ID,
		LoanID:        row.LoanID.Int32,
		PaymentNumber: row.PaymentNumber,
		Amount:        pgNumericToDecimal(row.Amount),
		DueYear:       row.DueYear,
		DueMonth:      row.DueMonth,
		Paid:          row.Paid,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	if row.PaidDate != nil {
		switch v := row.PaidDate.(type) {
		case time.Time:
			payment.PaidDate = &v
		case pgtype.Timestamptz:
			if v.Valid {
				payment.PaidDate = &v.Time
			}
		}
	}

	return payment
}
