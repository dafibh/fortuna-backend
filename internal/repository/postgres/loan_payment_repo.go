package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// LoanPaymentRepository implements domain.LoanPaymentRepository using PostgreSQL
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

// Create creates a new loan payment
func (r *LoanPaymentRepository) Create(payment *domain.LoanPayment) (*domain.LoanPayment, error) {
	ctx := context.Background()

	amount, err := decimalToPgNumeric(payment.Amount)
	if err != nil {
		return nil, err
	}

	paidDate := pgtype.Date{}
	if payment.PaidDate != nil {
		paidDate.Time = *payment.PaidDate
		paidDate.Valid = true
	}

	created, err := r.queries.CreateLoanPayment(ctx, sqlc.CreateLoanPaymentParams{
		LoanID:        payment.LoanID,
		PaymentNumber: payment.PaymentNumber,
		Amount:        amount,
		DueYear:       payment.DueYear,
		DueMonth:      payment.DueMonth,
		Paid:          payment.Paid,
		PaidDate:      paidDate,
	})
	if err != nil {
		return nil, err
	}
	return sqlcLoanPaymentToDomain(created), nil
}

// CreateBatch creates multiple loan payments efficiently
func (r *LoanPaymentRepository) CreateBatch(payments []*domain.LoanPayment) error {
	ctx := context.Background()

	// Use a transaction for batch insert
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	for _, payment := range payments {
		amount, err := decimalToPgNumeric(payment.Amount)
		if err != nil {
			return err
		}

		paidDate := pgtype.Date{}
		if payment.PaidDate != nil {
			paidDate.Time = *payment.PaidDate
			paidDate.Valid = true
		}

		_, err = qtx.CreateLoanPayment(ctx, sqlc.CreateLoanPaymentParams{
			LoanID:        payment.LoanID,
			PaymentNumber: payment.PaymentNumber,
			Amount:        amount,
			DueYear:       payment.DueYear,
			DueMonth:      payment.DueMonth,
			Paid:          payment.Paid,
			PaidDate:      paidDate,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// CreateBatchTx creates multiple loan payments within an existing transaction
func (r *LoanPaymentRepository) CreateBatchTx(tx any, payments []*domain.LoanPayment) error {
	ctx := context.Background()
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("invalid transaction type")
	}
	qtx := r.queries.WithTx(pgxTx)

	for _, payment := range payments {
		amount, err := decimalToPgNumeric(payment.Amount)
		if err != nil {
			return err
		}

		paidDate := pgtype.Date{}
		if payment.PaidDate != nil {
			paidDate.Time = *payment.PaidDate
			paidDate.Valid = true
		}

		_, err = qtx.CreateLoanPayment(ctx, sqlc.CreateLoanPaymentParams{
			LoanID:        payment.LoanID,
			PaymentNumber: payment.PaymentNumber,
			Amount:        amount,
			DueYear:       payment.DueYear,
			DueMonth:      payment.DueMonth,
			Paid:          payment.Paid,
			PaidDate:      paidDate,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByID retrieves a loan payment by its ID
func (r *LoanPaymentRepository) GetByID(id int32) (*domain.LoanPayment, error) {
	ctx := context.Background()
	payment, err := r.queries.GetLoanPaymentByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanPaymentNotFound
		}
		return nil, err
	}
	return sqlcLoanPaymentToDomain(payment), nil
}

// GetByLoanID retrieves all payments for a loan
func (r *LoanPaymentRepository) GetByLoanID(loanID int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()
	payments, err := r.queries.GetLoanPaymentsByLoanID(ctx, loanID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanPayment, len(payments))
	for i, p := range payments {
		result[i] = sqlcLoanPaymentToDomain(p)
	}
	return result, nil
}

// GetByLoanAndNumber retrieves a specific payment by loan ID and payment number
func (r *LoanPaymentRepository) GetByLoanAndNumber(loanID int32, paymentNumber int32) (*domain.LoanPayment, error) {
	ctx := context.Background()
	payment, err := r.queries.GetLoanPaymentByLoanAndNumber(ctx, sqlc.GetLoanPaymentByLoanAndNumberParams{
		LoanID:        loanID,
		PaymentNumber: paymentNumber,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanPaymentNotFound
		}
		return nil, err
	}
	return sqlcLoanPaymentToDomain(payment), nil
}

// UpdateAmount updates the amount of a specific payment
func (r *LoanPaymentRepository) UpdateAmount(id int32, amount decimal.Decimal) (*domain.LoanPayment, error) {
	ctx := context.Background()

	pgAmount, err := decimalToPgNumeric(amount)
	if err != nil {
		return nil, err
	}

	updated, err := r.queries.UpdateLoanPaymentAmount(ctx, sqlc.UpdateLoanPaymentAmountParams{
		ID:     id,
		Amount: pgAmount,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanPaymentNotFound
		}
		return nil, err
	}
	return sqlcLoanPaymentToDomain(updated), nil
}

// TogglePaid toggles the paid status of a payment
func (r *LoanPaymentRepository) TogglePaid(id int32, paid bool, paidDate *time.Time) (*domain.LoanPayment, error) {
	ctx := context.Background()

	pgPaidDate := pgtype.Date{}
	if paidDate != nil {
		pgPaidDate.Time = *paidDate
		pgPaidDate.Valid = true
	}

	updated, err := r.queries.ToggleLoanPaymentPaid(ctx, sqlc.ToggleLoanPaymentPaidParams{
		ID:       id,
		Paid:     paid,
		PaidDate: pgPaidDate,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanPaymentNotFound
		}
		return nil, err
	}
	return sqlcLoanPaymentToDomain(updated), nil
}

// GetByMonth retrieves all loan payments due in a specific month
func (r *LoanPaymentRepository) GetByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	ctx := context.Background()
	payments, err := r.queries.GetLoanPaymentsByMonth(ctx, sqlc.GetLoanPaymentsByMonthParams{
		WorkspaceID: workspaceID,
		DueYear:     int32(year),
		DueMonth:    int32(month),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanPayment, len(payments))
	for i, p := range payments {
		result[i] = sqlcLoanPaymentToDomain(p)
	}
	return result, nil
}

// GetUnpaidByMonth retrieves unpaid loan payments due in a specific month
func (r *LoanPaymentRepository) GetUnpaidByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	ctx := context.Background()
	payments, err := r.queries.GetUnpaidLoanPaymentsByMonth(ctx, sqlc.GetUnpaidLoanPaymentsByMonthParams{
		WorkspaceID: workspaceID,
		DueYear:     int32(year),
		DueMonth:    int32(month),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanPayment, len(payments))
	for i, p := range payments {
		result[i] = sqlcLoanPaymentToDomain(p)
	}
	return result, nil
}

// GetDeleteStats retrieves payment statistics for a loan (used for delete confirmation)
func (r *LoanPaymentRepository) GetDeleteStats(loanID int32) (*domain.LoanDeleteStats, error) {
	ctx := context.Background()
	stats, err := r.queries.GetLoanDeleteStats(ctx, loanID)
	if err != nil {
		return nil, err
	}
	return &domain.LoanDeleteStats{
		TotalCount:  stats.TotalCount,
		PaidCount:   stats.PaidCount,
		UnpaidCount: stats.UnpaidCount,
		TotalAmount: pgNumericToDecimal(stats.TotalAmount),
	}, nil
}

// GetPaymentsWithDetailsByMonth retrieves loan payments with loan details for a specific month
func (r *LoanPaymentRepository) GetPaymentsWithDetailsByMonth(workspaceID int32, year, month int) ([]*domain.MonthlyPaymentDetail, error) {
	ctx := context.Background()
	payments, err := r.queries.GetLoanPaymentsWithDetailsByMonth(ctx, sqlc.GetLoanPaymentsWithDetailsByMonthParams{
		WorkspaceID: workspaceID,
		DueYear:     int32(year),
		DueMonth:    int32(month),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.MonthlyPaymentDetail, len(payments))
	for i, p := range payments {
		result[i] = &domain.MonthlyPaymentDetail{
			ID:            p.ID,
			LoanID:        p.LoanID,
			ItemName:      p.ItemName,
			PaymentNumber: p.PaymentNumber,
			TotalPayments: p.TotalPayments,
			Amount:        pgNumericToDecimal(p.Amount),
			Paid:          p.Paid,
		}
	}
	return result, nil
}

// SumUnpaidByMonth returns the total amount of unpaid loan payments for a specific month
func (r *LoanPaymentRepository) SumUnpaidByMonth(workspaceID int32, year, month int) (decimal.Decimal, error) {
	ctx := context.Background()
	total, err := r.queries.SumUnpaidLoanPaymentsByMonth(ctx, sqlc.SumUnpaidLoanPaymentsByMonthParams{
		WorkspaceID: workspaceID,
		DueYear:     int32(year),
		DueMonth:    int32(month),
	})
	if err != nil {
		return decimal.Zero, err
	}
	return pgNumericToDecimal(total), nil
}

// GetEarliestUnpaidMonth returns the earliest unpaid month for a provider
// Used for sequential enforcement in consolidated monthly payment mode
// Returns nil if no unpaid months exist (all payments complete)
func (r *LoanPaymentRepository) GetEarliestUnpaidMonth(workspaceID int32, providerID int32) (*domain.EarliestUnpaidMonth, error) {
	ctx := context.Background()
	row, err := r.queries.GetEarliestUnpaidMonth(ctx, sqlc.GetEarliestUnpaidMonthParams{
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
		Year:  row.DueYear,
		Month: row.DueMonth,
	}, nil
}

// GetUnpaidPaymentsByProviderMonth returns unpaid payments for a specific provider and month
// Used for Pay Month action in consolidated monthly mode
func (r *LoanPaymentRepository) GetUnpaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()
	payments, err := r.queries.GetUnpaidPaymentsByProviderMonth(ctx, sqlc.GetUnpaidPaymentsByProviderMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		DueYear:     year,
		DueMonth:    month,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanPayment, len(payments))
	for i, p := range payments {
		result[i] = sqlcLoanPaymentToDomain(p)
	}
	return result, nil
}

// BatchUpdatePaidTx atomically marks multiple payments as paid within an existing transaction
// Returns the count of updated rows and total amount
func (r *LoanPaymentRepository) BatchUpdatePaidTx(tx any, paymentIDs []int32, workspaceID int32) (int, decimal.Decimal, error) {
	ctx := context.Background()
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return 0, decimal.Zero, errors.New("invalid transaction type")
	}
	qtx := r.queries.WithTx(pgxTx)

	// First get the sum of amounts
	total, err := qtx.SumPaymentAmountsByIDs(ctx, sqlc.SumPaymentAmountsByIDsParams{
		Column1:     paymentIDs,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return 0, decimal.Zero, err
	}

	// Batch update
	now := time.Now()
	paidDate := pgtype.Date{
		Time:  now,
		Valid: true,
	}
	count, err := qtx.BatchUpdatePaid(ctx, sqlc.BatchUpdatePaidParams{
		Column1:  paymentIDs,
		PaidDate: paidDate,
	})
	if err != nil {
		return 0, decimal.Zero, err
	}

	return int(count), pgNumericToDecimal(total), nil
}

// GetLatestPaidMonth returns the latest paid month for a provider
// Used for reverse sequential enforcement in unpay action
// Returns nil if no paid months exist
func (r *LoanPaymentRepository) GetLatestPaidMonth(workspaceID int32, providerID int32) (*domain.LatestPaidMonth, error) {
	ctx := context.Background()
	row, err := r.queries.GetLatestPaidMonth(ctx, sqlc.GetLatestPaidMonthParams{
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
		Year:  row.DueYear,
		Month: row.DueMonth,
	}, nil
}

// GetPaidPaymentsByProviderMonth returns paid payments for a specific provider and month
// Used for Unpay Month action in consolidated monthly mode
func (r *LoanPaymentRepository) GetPaidPaymentsByProviderMonth(workspaceID int32, providerID int32, year int32, month int32) ([]*domain.LoanPayment, error) {
	ctx := context.Background()
	payments, err := r.queries.GetPaidPaymentsByProviderMonth(ctx, sqlc.GetPaidPaymentsByProviderMonthParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
		DueYear:     year,
		DueMonth:    month,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanPayment, len(payments))
	for i, p := range payments {
		result[i] = sqlcLoanPaymentToDomain(p)
	}
	return result, nil
}

// BatchUpdateUnpaidTx atomically marks multiple payments as unpaid within an existing transaction
// Returns the count of updated rows
func (r *LoanPaymentRepository) BatchUpdateUnpaidTx(tx any, paymentIDs []int32) (int, error) {
	ctx := context.Background()
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return 0, errors.New("invalid transaction type")
	}
	qtx := r.queries.WithTx(pgxTx)

	count, err := qtx.BatchUpdateUnpaid(ctx, paymentIDs)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// GetTrendRaw retrieves raw trend data (monthly aggregates by provider) for visualization
// Returns rows from startYear/startMonth onward, grouped by month and provider
// Gap months (months with no payments) are handled by the service layer
func (r *LoanPaymentRepository) GetTrendRaw(workspaceID int32, startYear int32, startMonth int32) ([]*domain.TrendRawRow, error) {
	ctx := context.Background()
	rows, err := r.queries.GetTrendByMonth(ctx, sqlc.GetTrendByMonthParams{
		WorkspaceID: workspaceID,
		DueYear:     startYear,
		DueMonth:    startMonth,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.TrendRawRow, len(rows))
	for i, row := range rows {
		result[i] = &domain.TrendRawRow{
			DueYear:      row.DueYear,
			DueMonth:     row.DueMonth,
			ProviderID:   row.ProviderID,
			ProviderName: row.ProviderName,
			Total:        pgNumericToDecimal(row.Total),
			IsPaid:       row.IsPaid,
		}
	}
	return result, nil
}

// Helper function to convert sqlc type to domain type
func sqlcLoanPaymentToDomain(lp sqlc.LoanPayment) *domain.LoanPayment {
	payment := &domain.LoanPayment{
		ID:            lp.ID,
		LoanID:        lp.LoanID,
		PaymentNumber: lp.PaymentNumber,
		Amount:        pgNumericToDecimal(lp.Amount),
		DueYear:       lp.DueYear,
		DueMonth:      lp.DueMonth,
		Paid:          lp.Paid,
		CreatedAt:     lp.CreatedAt.Time,
		UpdatedAt:     lp.UpdatedAt.Time,
	}

	if lp.PaidDate.Valid {
		paidDate := lp.PaidDate.Time
		payment.PaidDate = &paidDate
	}

	return payment
}
