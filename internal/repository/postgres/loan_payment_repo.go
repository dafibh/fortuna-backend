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
func (r *LoanPaymentRepository) CreateBatchTx(tx interface{}, payments []*domain.LoanPayment) error {
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
