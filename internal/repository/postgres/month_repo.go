package postgres

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

// MonthRepository implements domain.MonthRepository using PostgreSQL
type MonthRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewMonthRepository creates a new MonthRepository
func NewMonthRepository(pool *pgxpool.Pool) *MonthRepository {
	return &MonthRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new month record
func (r *MonthRepository) Create(workspaceID int32, year, month int, startDate, endDate time.Time, startingBalance decimal.Decimal) (*domain.Month, error) {
	ctx := context.Background()

	startingBalanceNum, err := decimalToPgNumeric(startingBalance)
	if err != nil {
		return nil, err
	}

	created, err := r.queries.CreateMonth(ctx, sqlc.CreateMonthParams{
		WorkspaceID:     workspaceID,
		Year:            int32(year),
		Month:           int32(month),
		StartDate:       timeToPgDate(startDate),
		EndDate:         timeToPgDate(endDate),
		StartingBalance: startingBalanceNum,
	})
	if err != nil {
		return nil, err
	}
	return sqlcMonthToDomain(created), nil
}

// GetByYearMonth retrieves a month by year and month number
func (r *MonthRepository) GetByYearMonth(workspaceID int32, year, month int) (*domain.Month, error) {
	ctx := context.Background()

	m, err := r.queries.GetMonthByYearMonth(ctx, sqlc.GetMonthByYearMonthParams{
		WorkspaceID: workspaceID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMonthNotFound
		}
		return nil, err
	}
	return sqlcMonthToDomain(m), nil
}

// GetLatest retrieves the most recent month for a workspace
func (r *MonthRepository) GetLatest(workspaceID int32) (*domain.Month, error) {
	ctx := context.Background()

	m, err := r.queries.GetLatestMonth(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMonthNotFound
		}
		return nil, err
	}
	return sqlcMonthToDomain(m), nil
}

// GetAll retrieves all months for a workspace
func (r *MonthRepository) GetAll(workspaceID int32) ([]*domain.Month, error) {
	ctx := context.Background()

	months, err := r.queries.GetAllMonths(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Month, len(months))
	for i, m := range months {
		result[i] = sqlcMonthToDomain(m)
	}
	return result, nil
}

// UpdateStartingBalance updates the starting balance of a month
func (r *MonthRepository) UpdateStartingBalance(workspaceID, id int32, balance decimal.Decimal) error {
	ctx := context.Background()

	balanceNum, err := decimalToPgNumeric(balance)
	if err != nil {
		return err
	}

	return r.queries.UpdateMonthStartingBalance(ctx, sqlc.UpdateMonthStartingBalanceParams{
		WorkspaceID:     workspaceID,
		ID:              id,
		StartingBalance: balanceNum,
	})
}

// Helper functions

func sqlcMonthToDomain(m sqlc.Month) *domain.Month {
	return &domain.Month{
		ID:              m.ID,
		WorkspaceID:     m.WorkspaceID,
		Year:            int(m.Year),
		Month:           int(m.Month),
		StartDate:       pgDateToTime(m.StartDate),
		EndDate:         pgDateToTime(m.EndDate),
		StartingBalance: pgNumericToDecimal(m.StartingBalance),
		CreatedAt:       m.CreatedAt.Time,
		UpdatedAt:       m.UpdatedAt.Time,
	}
}

func timeToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}

func pgDateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
