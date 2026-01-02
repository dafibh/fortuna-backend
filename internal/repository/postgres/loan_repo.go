package postgres

import (
	"context"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoanRepository implements domain.LoanRepository using PostgreSQL
type LoanRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewLoanRepository creates a new LoanRepository
func NewLoanRepository(pool *pgxpool.Pool) *LoanRepository {
	return &LoanRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new loan
func (r *LoanRepository) Create(loan *domain.Loan) (*domain.Loan, error) {
	ctx := context.Background()

	totalAmount, err := decimalToPgNumeric(loan.TotalAmount)
	if err != nil {
		return nil, err
	}

	interestRate, err := decimalToPgNumeric(loan.InterestRate)
	if err != nil {
		return nil, err
	}

	monthlyPayment, err := decimalToPgNumeric(loan.MonthlyPayment)
	if err != nil {
		return nil, err
	}

	purchaseDate := pgtype.Date{
		Time:  loan.PurchaseDate,
		Valid: true,
	}

	notes := pgtype.Text{}
	if loan.Notes != nil {
		notes.String = *loan.Notes
		notes.Valid = true
	}

	created, err := r.queries.CreateLoan(ctx, sqlc.CreateLoanParams{
		WorkspaceID:       loan.WorkspaceID,
		ProviderID:        loan.ProviderID,
		ItemName:          loan.ItemName,
		TotalAmount:       totalAmount,
		NumMonths:         loan.NumMonths,
		PurchaseDate:      purchaseDate,
		InterestRate:      interestRate,
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  loan.FirstPaymentYear,
		FirstPaymentMonth: loan.FirstPaymentMonth,
		Notes:             notes,
	})
	if err != nil {
		return nil, err
	}
	return sqlcLoanToDomain(created), nil
}

// GetByID retrieves a loan by its ID within a workspace
func (r *LoanRepository) GetByID(workspaceID int32, id int32) (*domain.Loan, error) {
	ctx := context.Background()
	loan, err := r.queries.GetLoanByID(ctx, sqlc.GetLoanByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanNotFound
		}
		return nil, err
	}
	return sqlcLoanToDomain(loan), nil
}

// GetAllByWorkspace retrieves all loans for a workspace
func (r *LoanRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.Loan, error) {
	ctx := context.Background()
	loans, err := r.queries.ListLoans(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Loan, len(loans))
	for i, l := range loans {
		result[i] = sqlcLoanToDomain(l)
	}
	return result, nil
}

// GetActiveByWorkspace retrieves active loans for a workspace
func (r *LoanRepository) GetActiveByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	ctx := context.Background()
	loans, err := r.queries.ListActiveLoans(ctx, sqlc.ListActiveLoansParams{
		WorkspaceID:       workspaceID,
		FirstPaymentYear:  int32(currentYear),
		FirstPaymentMonth: int32(currentMonth),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Loan, len(loans))
	for i, l := range loans {
		result[i] = sqlcLoanToDomain(l)
	}
	return result, nil
}

// GetCompletedByWorkspace retrieves completed loans for a workspace
func (r *LoanRepository) GetCompletedByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	ctx := context.Background()
	loans, err := r.queries.ListCompletedLoans(ctx, sqlc.ListCompletedLoansParams{
		WorkspaceID:       workspaceID,
		FirstPaymentYear:  int32(currentYear),
		FirstPaymentMonth: int32(currentMonth),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Loan, len(loans))
	for i, l := range loans {
		result[i] = sqlcLoanToDomain(l)
	}
	return result, nil
}

// Update updates a loan
func (r *LoanRepository) Update(loan *domain.Loan) (*domain.Loan, error) {
	ctx := context.Background()

	totalAmount, err := decimalToPgNumeric(loan.TotalAmount)
	if err != nil {
		return nil, err
	}

	interestRate, err := decimalToPgNumeric(loan.InterestRate)
	if err != nil {
		return nil, err
	}

	monthlyPayment, err := decimalToPgNumeric(loan.MonthlyPayment)
	if err != nil {
		return nil, err
	}

	purchaseDate := pgtype.Date{
		Time:  loan.PurchaseDate,
		Valid: true,
	}

	notes := pgtype.Text{}
	if loan.Notes != nil {
		notes.String = *loan.Notes
		notes.Valid = true
	}

	updated, err := r.queries.UpdateLoan(ctx, sqlc.UpdateLoanParams{
		ID:                loan.ID,
		WorkspaceID:       loan.WorkspaceID,
		ItemName:          loan.ItemName,
		TotalAmount:       totalAmount,
		NumMonths:         loan.NumMonths,
		PurchaseDate:      purchaseDate,
		InterestRate:      interestRate,
		MonthlyPayment:    monthlyPayment,
		FirstPaymentYear:  loan.FirstPaymentYear,
		FirstPaymentMonth: loan.FirstPaymentMonth,
		Notes:             notes,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanNotFound
		}
		return nil, err
	}
	return sqlcLoanToDomain(updated), nil
}

// SoftDelete marks a loan as deleted
func (r *LoanRepository) SoftDelete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteLoan(ctx, sqlc.DeleteLoanParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// CountActiveLoansByProvider counts active loans for a provider
func (r *LoanRepository) CountActiveLoansByProvider(workspaceID int32, providerID int32, currentYear, currentMonth int) (int64, error) {
	ctx := context.Background()
	return r.queries.CountActiveLoansByProvider(ctx, sqlc.CountActiveLoansByProviderParams{
		ProviderID:        providerID,
		WorkspaceID:       workspaceID,
		FirstPaymentYear:  int32(currentYear),
		FirstPaymentMonth: int32(currentMonth),
	})
}

// Helper functions

func sqlcLoanToDomain(l sqlc.Loan) *domain.Loan {
	loan := &domain.Loan{
		ID:                l.ID,
		WorkspaceID:       l.WorkspaceID,
		ProviderID:        l.ProviderID,
		ItemName:          l.ItemName,
		TotalAmount:       pgNumericToDecimal(l.TotalAmount),
		NumMonths:         l.NumMonths,
		InterestRate:      pgNumericToDecimal(l.InterestRate),
		MonthlyPayment:    pgNumericToDecimal(l.MonthlyPayment),
		FirstPaymentYear:  l.FirstPaymentYear,
		FirstPaymentMonth: l.FirstPaymentMonth,
		CreatedAt:         l.CreatedAt.Time,
		UpdatedAt:         l.UpdatedAt.Time,
	}

	// Handle purchase date
	if l.PurchaseDate.Valid {
		loan.PurchaseDate = l.PurchaseDate.Time
	} else {
		loan.PurchaseDate = time.Time{}
	}

	// Handle notes
	if l.Notes.Valid {
		loan.Notes = &l.Notes.String
	}

	// Handle deleted at
	if l.DeletedAt.Valid {
		loan.DeletedAt = &l.DeletedAt.Time
	}

	return loan
}
