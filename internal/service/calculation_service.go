package service

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculationService handles balance calculation logic
type CalculationService struct {
	accountRepo     domain.AccountRepository
	transactionRepo domain.TransactionRepository
}

// NewCalculationService creates a new CalculationService
func NewCalculationService(accountRepo domain.AccountRepository, transactionRepo domain.TransactionRepository) *CalculationService {
	return &CalculationService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// AccountBalanceResult holds calculated balance information for an account
type AccountBalanceResult struct {
	AccountID         int32
	InitialBalance    decimal.Decimal
	CalculatedBalance decimal.Decimal
	CCOutstanding     decimal.Decimal
}

// CalculateAccountBalances calculates balances for all accounts in a workspace
func (s *CalculationService) CalculateAccountBalances(workspaceID int32) (map[int32]*AccountBalanceResult, error) {
	// Get all accounts
	accounts, err := s.accountRepo.GetAllByWorkspace(workspaceID, false)
	if err != nil {
		return nil, err
	}

	// Get transaction summaries
	summaries, err := s.transactionRepo.GetAccountTransactionSummaries(workspaceID)
	if err != nil {
		return nil, err
	}

	// Build summary map
	summaryMap := make(map[int32]*domain.TransactionSummary)
	for _, summary := range summaries {
		summaryMap[summary.AccountID] = summary
	}

	// Calculate balances
	results := make(map[int32]*AccountBalanceResult)
	for _, account := range accounts {
		summary := summaryMap[account.ID]

		result := &AccountBalanceResult{
			AccountID:      account.ID,
			InitialBalance: account.InitialBalance,
		}

		if summary != nil {
			// calculated_balance = initial + income - expenses
			// For CC accounts, use ALL expenses (isPaid means "settled with bank", not "purchase happened")
			// For regular accounts, only count paid expenses
			if account.Template == domain.TemplateCreditCard {
				result.CalculatedBalance = account.InitialBalance.
					Add(summary.SumIncome).
					Sub(summary.SumAllExpenses)
				result.CCOutstanding = summary.SumUnpaidExpenses
			} else {
				result.CalculatedBalance = account.InitialBalance.
					Add(summary.SumIncome).
					Sub(summary.SumExpenses)
			}
		} else {
			// No transactions, balance = initial
			result.CalculatedBalance = account.InitialBalance
		}

		results[account.ID] = result
	}

	return results, nil
}

// CalculateAccountBalance calculates the balance for a single account
func (s *CalculationService) CalculateAccountBalance(workspaceID, accountID int32) (*AccountBalanceResult, error) {
	// Get the account
	account, err := s.accountRepo.GetByID(workspaceID, accountID)
	if err != nil {
		return nil, err
	}

	// Get all summaries and find the one for this account
	summaries, err := s.transactionRepo.GetAccountTransactionSummaries(workspaceID)
	if err != nil {
		return nil, err
	}

	var summary *domain.TransactionSummary
	for _, s := range summaries {
		if s.AccountID == accountID {
			summary = s
			break
		}
	}

	result := &AccountBalanceResult{
		AccountID:      account.ID,
		InitialBalance: account.InitialBalance,
	}

	if summary != nil {
		// For CC accounts, use ALL expenses (isPaid means "settled with bank", not "purchase happened")
		// For regular accounts, only count paid expenses
		if account.Template == domain.TemplateCreditCard {
			result.CalculatedBalance = account.InitialBalance.
				Add(summary.SumIncome).
				Sub(summary.SumAllExpenses)
			result.CCOutstanding = summary.SumUnpaidExpenses
		} else {
			result.CalculatedBalance = account.InitialBalance.
				Add(summary.SumIncome).
				Sub(summary.SumExpenses)
		}
	} else {
		result.CalculatedBalance = account.InitialBalance
	}

	return result, nil
}
