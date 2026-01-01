package service

import (
	"sort"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CCService handles credit card related business logic
type CCService struct {
	transactionRepo domain.TransactionRepository
	accountRepo     domain.AccountRepository
}

// NewCCService creates a new CCService
func NewCCService(transactionRepo domain.TransactionRepository, accountRepo domain.AccountRepository) *CCService {
	return &CCService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
	}
}

// CreateCCPayment creates a CC payment transaction (income on CC account)
// Optionally creates a linked expense on the source bank account
func (s *CCService) CreateCCPayment(workspaceID int32, req *domain.CreateCCPaymentRequest) (*domain.CCPaymentResponse, error) {
	// Validate CC account exists and is a credit card
	ccAccount, err := s.accountRepo.GetByID(workspaceID, req.CCAccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	if ccAccount.Template != domain.TemplateCreditCard {
		return nil, domain.ErrInvalidAccountType
	}

	// Validate amount is positive
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Validate source account if provided
	if req.SourceAccountID != nil {
		sourceAccount, err := s.accountRepo.GetByID(workspaceID, *req.SourceAccountID)
		if err != nil {
			return nil, domain.ErrAccountNotFound
		}
		// Source cannot be a credit card
		if sourceAccount.Template == domain.TemplateCreditCard {
			return nil, domain.ErrInvalidSourceAccount
		}
	}

	// Build notes
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	response := &domain.CCPaymentResponse{}

	// If source account provided, create linked transfer pair
	if req.SourceAccountID != nil {
		transferPairID := uuid.New()

		// CC income transaction (reduces outstanding balance)
		ccTx := &domain.Transaction{
			WorkspaceID:     workspaceID,
			AccountID:       req.CCAccountID,
			Name:            "CC Payment",
			Amount:          req.Amount,
			Type:            domain.TransactionTypeIncome,
			TransactionDate: req.TransactionDate,
			IsPaid:          true,
			Notes:           notes,
			TransferPairID:  &transferPairID,
			IsCCPayment:     true,
		}

		// Source expense transaction
		sourceTx := &domain.Transaction{
			WorkspaceID:     workspaceID,
			AccountID:       *req.SourceAccountID,
			Name:            "CC Payment",
			Amount:          req.Amount,
			Type:            domain.TransactionTypeExpense,
			TransactionDate: req.TransactionDate,
			IsPaid:          true,
			Notes:           notes,
			TransferPairID:  &transferPairID,
			IsCCPayment:     false,
		}

		result, err := s.transactionRepo.CreateTransferPair(ccTx, sourceTx)
		if err != nil {
			return nil, err
		}

		response.CCTransaction = result.FromTransaction
		response.SourceTransaction = result.ToTransaction
	} else {
		// No source account - create single CC income transaction
		ccTx := &domain.Transaction{
			WorkspaceID:     workspaceID,
			AccountID:       req.CCAccountID,
			Name:            "CC Payment",
			Amount:          req.Amount,
			Type:            domain.TransactionTypeIncome,
			TransactionDate: req.TransactionDate,
			IsPaid:          true,
			Notes:           notes,
			IsCCPayment:     true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		created, err := s.transactionRepo.Create(ccTx)
		if err != nil {
			return nil, err
		}

		response.CCTransaction = created
	}

	return response, nil
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
