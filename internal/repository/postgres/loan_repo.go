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
	return r.createLoan(ctx, r.queries, loan)
}

// CreateTx creates a new loan within a transaction
func (r *LoanRepository) CreateTx(tx interface{}, loan *domain.Loan) (*domain.Loan, error) {
	ctx := context.Background()
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("invalid transaction type")
	}
	qtx := r.queries.WithTx(pgxTx)
	return r.createLoan(ctx, qtx, loan)
}

// createLoan is the internal implementation for creating a loan
func (r *LoanRepository) createLoan(ctx context.Context, q *sqlc.Queries, loan *domain.Loan) (*domain.Loan, error) {
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

	accountID := pgtype.Int4{}
	if loan.AccountID > 0 {
		accountID.Int32 = loan.AccountID
		accountID.Valid = true
	}

	settlementIntent := pgtype.Text{}
	if loan.SettlementIntent != nil {
		settlementIntent.String = *loan.SettlementIntent
		settlementIntent.Valid = true
	}

	created, err := q.CreateLoan(ctx, sqlc.CreateLoanParams{
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
		AccountID:         accountID,
		SettlementIntent:  settlementIntent,
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

// UpdatePartial updates only the editable fields (item_name, notes) of a loan
func (r *LoanRepository) UpdatePartial(workspaceID int32, id int32, itemName string, notes *string) (*domain.Loan, error) {
	ctx := context.Background()

	pgNotes := pgtype.Text{}
	if notes != nil {
		pgNotes.String = *notes
		pgNotes.Valid = true
	}

	updated, err := r.queries.UpdateLoanPartial(ctx, sqlc.UpdateLoanPartialParams{
		ID:          id,
		WorkspaceID: workspaceID,
		ItemName:    itemName,
		Notes:       pgNotes,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLoanNotFound
		}
		return nil, err
	}
	return sqlcLoanToDomain(updated), nil
}

// UpdateEditableFields updates item name, provider, and notes
// Provider can only change if no payments have been made (validated at service layer)
func (r *LoanRepository) UpdateEditableFields(workspaceID int32, id int32, itemName string, providerID int32, notes *string) (*domain.Loan, error) {
	ctx := context.Background()

	pgNotes := pgtype.Text{}
	if notes != nil {
		pgNotes.String = *notes
		pgNotes.Valid = true
	}

	updated, err := r.queries.UpdateLoanEditableFields(ctx, sqlc.UpdateLoanEditableFieldsParams{
		ID:          id,
		WorkspaceID: workspaceID,
		ItemName:    itemName,
		ProviderID:  providerID,
		Notes:       pgNotes,
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

// GetAllWithStats retrieves all loans with payment statistics
// CL v2: Uses transactions with loan_id instead of loan_payments table
func (r *LoanRepository) GetAllWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	ctx := context.Background()
	rows, err := r.queries.GetLoansWithStats(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanWithStats, len(rows))
	for i, row := range rows {
		result[i] = sqlcLoansWithStatsRowToDomain(row)
	}
	return result, nil
}

// GetActiveWithStats retrieves active loans (remaining_balance > 0) with payment statistics
// CL v2: Uses transactions with loan_id instead of loan_payments table
func (r *LoanRepository) GetActiveWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	ctx := context.Background()
	rows, err := r.queries.GetActiveLoansWithStats(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanWithStats, len(rows))
	for i, row := range rows {
		result[i] = sqlcActiveLoansWithStatsRowToDomain(row)
	}
	return result, nil
}

// GetCompletedWithStats retrieves completed loans (remaining_balance = 0) with payment statistics
// CL v2: Uses transactions with loan_id instead of loan_payments table
func (r *LoanRepository) GetCompletedWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	ctx := context.Background()
	rows, err := r.queries.GetCompletedLoansWithStats(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanWithStats, len(rows))
	for i, row := range rows {
		result[i] = sqlcCompletedLoansWithStatsRowToDomain(row)
	}
	return result, nil
}

// GetByProviderWithStats retrieves all loans for a provider with payment statistics
// CL v2: Uses transactions with loan_id for stats, ordered by unpaid first then item name
func (r *LoanRepository) GetByProviderWithStats(workspaceID int32, providerID int32) ([]*domain.LoanWithStats, error) {
	ctx := context.Background()
	rows, err := r.queries.GetLoansWithStatsByProvider(ctx, sqlc.GetLoansWithStatsByProviderParams{
		WorkspaceID: workspaceID,
		ProviderID:  providerID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoanWithStats, len(rows))
	for i, row := range rows {
		result[i] = sqlcLoansWithStatsByProviderRowToDomain(row)
	}
	return result, nil
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

	// Handle account ID
	if l.AccountID.Valid {
		loan.AccountID = l.AccountID.Int32
	}

	// Handle settlement intent
	if l.SettlementIntent.Valid {
		loan.SettlementIntent = &l.SettlementIntent.String
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

// CL v2: Helper functions for converting stats rows from transaction-based queries

func sqlcLoansWithStatsRowToDomain(row sqlc.GetLoansWithStatsRow) *domain.LoanWithStats {
	loan := &domain.LoanWithStats{
		Loan: domain.Loan{
			ID:                row.ID,
			WorkspaceID:       row.WorkspaceID,
			ProviderID:        row.ProviderID,
			ItemName:          row.ItemName,
			TotalAmount:       pgNumericToDecimal(row.TotalAmount),
			NumMonths:         row.NumMonths,
			InterestRate:      pgNumericToDecimal(row.InterestRate),
			MonthlyPayment:    pgNumericToDecimal(row.MonthlyPayment),
			FirstPaymentYear:  row.FirstPaymentYear,
			FirstPaymentMonth: row.FirstPaymentMonth,
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
		},
		LastPaymentYear:  row.LastPaymentYear,
		LastPaymentMonth: row.LastPaymentMonth,
		TotalCount:       row.TotalCount,
		PaidCount:        row.PaidCount,
		RemainingBalance: pgNumericToDecimal(row.RemainingBalance),
	}

	// Handle optional fields
	if row.PurchaseDate.Valid {
		loan.PurchaseDate = row.PurchaseDate.Time
	}
	if row.AccountID.Valid {
		loan.AccountID = row.AccountID.Int32
	}
	if row.SettlementIntent.Valid {
		loan.SettlementIntent = &row.SettlementIntent.String
	}
	if row.Notes.Valid {
		loan.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		loan.DeletedAt = &row.DeletedAt.Time
	}

	// Calculate progress percentage
	if loan.TotalCount > 0 {
		loan.Progress = float64(loan.PaidCount) / float64(loan.TotalCount) * 100
	}

	return loan
}

func sqlcActiveLoansWithStatsRowToDomain(row sqlc.GetActiveLoansWithStatsRow) *domain.LoanWithStats {
	loan := &domain.LoanWithStats{
		Loan: domain.Loan{
			ID:                row.ID,
			WorkspaceID:       row.WorkspaceID,
			ProviderID:        row.ProviderID,
			ItemName:          row.ItemName,
			TotalAmount:       pgNumericToDecimal(row.TotalAmount),
			NumMonths:         row.NumMonths,
			InterestRate:      pgNumericToDecimal(row.InterestRate),
			MonthlyPayment:    pgNumericToDecimal(row.MonthlyPayment),
			FirstPaymentYear:  row.FirstPaymentYear,
			FirstPaymentMonth: row.FirstPaymentMonth,
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
		},
		LastPaymentYear:  row.LastPaymentYear,
		LastPaymentMonth: row.LastPaymentMonth,
		TotalCount:       row.TotalCount,
		PaidCount:        row.PaidCount,
		RemainingBalance: pgNumericToDecimal(row.RemainingBalance),
	}

	if row.PurchaseDate.Valid {
		loan.PurchaseDate = row.PurchaseDate.Time
	}
	if row.AccountID.Valid {
		loan.AccountID = row.AccountID.Int32
	}
	if row.SettlementIntent.Valid {
		loan.SettlementIntent = &row.SettlementIntent.String
	}
	if row.Notes.Valid {
		loan.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		loan.DeletedAt = &row.DeletedAt.Time
	}
	if loan.TotalCount > 0 {
		loan.Progress = float64(loan.PaidCount) / float64(loan.TotalCount) * 100
	}

	return loan
}

func sqlcCompletedLoansWithStatsRowToDomain(row sqlc.GetCompletedLoansWithStatsRow) *domain.LoanWithStats {
	loan := &domain.LoanWithStats{
		Loan: domain.Loan{
			ID:                row.ID,
			WorkspaceID:       row.WorkspaceID,
			ProviderID:        row.ProviderID,
			ItemName:          row.ItemName,
			TotalAmount:       pgNumericToDecimal(row.TotalAmount),
			NumMonths:         row.NumMonths,
			InterestRate:      pgNumericToDecimal(row.InterestRate),
			MonthlyPayment:    pgNumericToDecimal(row.MonthlyPayment),
			FirstPaymentYear:  row.FirstPaymentYear,
			FirstPaymentMonth: row.FirstPaymentMonth,
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
		},
		LastPaymentYear:  row.LastPaymentYear,
		LastPaymentMonth: row.LastPaymentMonth,
		TotalCount:       row.TotalCount,
		PaidCount:        row.PaidCount,
		RemainingBalance: pgNumericToDecimal(row.RemainingBalance),
	}

	if row.PurchaseDate.Valid {
		loan.PurchaseDate = row.PurchaseDate.Time
	}
	if row.AccountID.Valid {
		loan.AccountID = row.AccountID.Int32
	}
	if row.SettlementIntent.Valid {
		loan.SettlementIntent = &row.SettlementIntent.String
	}
	if row.Notes.Valid {
		loan.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		loan.DeletedAt = &row.DeletedAt.Time
	}
	if loan.TotalCount > 0 {
		loan.Progress = float64(loan.PaidCount) / float64(loan.TotalCount) * 100
	}

	return loan
}

func sqlcLoansWithStatsByProviderRowToDomain(row sqlc.GetLoansWithStatsByProviderRow) *domain.LoanWithStats {
	loan := &domain.LoanWithStats{
		Loan: domain.Loan{
			ID:                row.ID,
			WorkspaceID:       row.WorkspaceID,
			ProviderID:        row.ProviderID,
			ItemName:          row.ItemName,
			TotalAmount:       pgNumericToDecimal(row.TotalAmount),
			NumMonths:         row.NumMonths,
			InterestRate:      pgNumericToDecimal(row.InterestRate),
			MonthlyPayment:    pgNumericToDecimal(row.MonthlyPayment),
			FirstPaymentYear:  row.FirstPaymentYear,
			FirstPaymentMonth: row.FirstPaymentMonth,
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
		},
		LastPaymentYear:  row.LastPaymentYear,
		LastPaymentMonth: row.LastPaymentMonth,
		TotalCount:       row.TotalCount,
		PaidCount:        row.PaidCount,
		RemainingBalance: pgNumericToDecimal(row.RemainingBalance),
	}

	if row.PurchaseDate.Valid {
		loan.PurchaseDate = row.PurchaseDate.Time
	}
	if row.AccountID.Valid {
		loan.AccountID = row.AccountID.Int32
	}
	if row.SettlementIntent.Valid {
		loan.SettlementIntent = &row.SettlementIntent.String
	}
	if row.Notes.Valid {
		loan.Notes = &row.Notes.String
	}
	if row.DeletedAt.Valid {
		loan.DeletedAt = &row.DeletedAt.Time
	}
	if loan.TotalCount > 0 {
		loan.Progress = float64(loan.PaidCount) / float64(loan.TotalCount) * 100
	}

	return loan
}
