package service

import (
	"sort"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// CCService handles credit card related business logic
type CCService struct {
	transactionRepo domain.TransactionRepository
}

// NewCCService creates a new CCService
func NewCCService(transactionRepo domain.TransactionRepository) *CCService {
	return &CCService{
		transactionRepo: transactionRepo,
	}
}

// GetPayableBreakdown returns CC transactions grouped by settlement intent and account
func (s *CCService) GetPayableBreakdown(workspaceID int32) (*domain.CCPayableBreakdown, error) {
	transactions, err := s.transactionRepo.GetCCPayableBreakdown(workspaceID)
	if err != nil {
		return nil, err
	}

	// Group by settlement intent, then by account
	thisMonthByAccount := make(map[int32]*domain.CCPayableByAccount)
	nextMonthByAccount := make(map[int32]*domain.CCPayableByAccount)

	for _, t := range transactions {
		target := thisMonthByAccount
		if t.SettlementIntent == domain.CCSettlementNextMonth {
			target = nextMonthByAccount
		}

		if _, exists := target[t.AccountID]; !exists {
			target[t.AccountID] = &domain.CCPayableByAccount{
				AccountID:    t.AccountID,
				AccountName:  t.AccountName,
				Total:        decimal.Zero,
				Transactions: []domain.CCPayableTransaction{},
			}
		}

		target[t.AccountID].Transactions = append(target[t.AccountID].Transactions, *t)
		target[t.AccountID].Total = target[t.AccountID].Total.Add(t.Amount)
	}

	// Convert maps to slices and calculate totals
	result := &domain.CCPayableBreakdown{
		ThisMonth:      mapToSlice(thisMonthByAccount),
		NextMonth:      mapToSlice(nextMonthByAccount),
		ThisMonthTotal: decimal.Zero,
		NextMonthTotal: decimal.Zero,
	}

	for _, acc := range result.ThisMonth {
		result.ThisMonthTotal = result.ThisMonthTotal.Add(acc.Total)
	}
	for _, acc := range result.NextMonth {
		result.NextMonthTotal = result.NextMonthTotal.Add(acc.Total)
	}
	result.GrandTotal = result.ThisMonthTotal.Add(result.NextMonthTotal)

	return result, nil
}

// mapToSlice converts a map of CCPayableByAccount to a sorted slice
func mapToSlice(m map[int32]*domain.CCPayableByAccount) []domain.CCPayableByAccount {
	result := make([]domain.CCPayableByAccount, 0, len(m))
	for _, v := range m {
		result = append(result, *v)
	}
	// Sort by account name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].AccountName < result[j].AccountName
	})
	return result
}
