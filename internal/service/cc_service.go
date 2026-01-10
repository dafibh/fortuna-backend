package service

import (
	"fmt"
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

	// Validate notes length
	if len(req.Notes) > domain.MaxTransactionNotesLength {
		return nil, domain.ErrNotesTooLong
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

// SettleCCTransactions atomically settles CC transactions and creates a transfer
func (s *CCService) SettleCCTransactions(workspaceID int32, req *domain.SettlementRequest) (*domain.SettlementResponse, error) {
	// Validate source account exists and is not a credit card
	sourceAccount, err := s.accountRepo.GetByID(workspaceID, req.SourceAccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	if sourceAccount.Template == domain.TemplateCreditCard {
		return nil, domain.ErrInvalidSourceAccount
	}

	// Validate target CC account exists and is a credit card
	ccAccount, err := s.accountRepo.GetByID(workspaceID, req.TargetCCAccountID)
	if err != nil {
		return nil, domain.ErrAccountNotFound
	}
	if ccAccount.Template != domain.TemplateCreditCard {
		return nil, domain.ErrInvalidAccountType
	}

	// Get all transactions to validate and calculate total
	transactions, err := s.transactionRepo.GetTransactionsByIDs(workspaceID, req.TransactionIDs)
	if err != nil {
		return nil, err
	}

	// Validate all transactions exist
	if len(transactions) != len(req.TransactionIDs) {
		return nil, domain.ErrTransactionNotFound
	}

	// Validate all transactions and calculate total amount
	totalAmount := decimal.Zero
	for _, tx := range transactions {
		// Must be billed CC transactions
		if tx.CCState == nil || *tx.CCState != domain.CCStateBilled {
			return nil, domain.ErrInvalidCCState
		}
		// Must be deferred
		if tx.CCSettlementIntent == nil || (*tx.CCSettlementIntent != domain.CCSettlementDeferred && *tx.CCSettlementIntent != domain.CCSettlementNextMonth) {
			return nil, domain.ErrInvalidSettlementIntent
		}
		// Must be for the target CC account
		if tx.AccountID != req.TargetCCAccountID {
			return nil, domain.ErrInvalidAccountType
		}
		totalAmount = totalAmount.Add(tx.Amount)
	}

	// Create the transfer (Bank expense → CC income)
	transferPairID := uuid.New()
	now := time.Now()

	// CC income transaction (reduces outstanding balance)
	ccTx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       req.TargetCCAccountID,
		Name:            "CC Settlement",
		Amount:          totalAmount,
		Type:            domain.TransactionTypeIncome,
		TransactionDate: now,
		IsPaid:          true,
		TransferPairID:  &transferPairID,
		IsCCPayment:     true,
	}

	// Source expense transaction
	sourceTx := &domain.Transaction{
		WorkspaceID:     workspaceID,
		AccountID:       req.SourceAccountID,
		Name:            "CC Settlement",
		Amount:          totalAmount,
		Type:            domain.TransactionTypeExpense,
		TransactionDate: now,
		IsPaid:          true,
		TransferPairID:  &transferPairID,
		IsCCPayment:     false,
	}

	transferResult, err := s.transactionRepo.CreateTransferPair(ccTx, sourceTx)
	if err != nil {
		return nil, err
	}

	// Bulk settle all the transactions
	settledCount, err := s.transactionRepo.BulkSettleTransactions(workspaceID, req.TransactionIDs)
	if err != nil {
		return nil, err
	}

	return &domain.SettlementResponse{
		TransferID:   transferResult.FromTransaction.ID,
		SettledCount: int(settledCount),
		TotalAmount:  totalAmount,
		SettledAt:    now,
	}, nil
}

// GetDeferredCCGroups returns deferred CC transactions grouped by origin month
func (s *CCService) GetDeferredCCGroups(workspaceID int32) ([]domain.DeferredGroup, error) {
	transactions, err := s.transactionRepo.GetDeferredCCByMonth(workspaceID)
	if err != nil {
		return nil, err
	}

	// Group by year-month
	groups := make(map[string]*domain.DeferredGroup)
	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	for _, tx := range transactions {
		key := fmt.Sprintf("%d-%02d", tx.OriginYear, tx.OriginMonth)

		if _, exists := groups[key]; !exists {
			monthDate := time.Date(tx.OriginYear, time.Month(tx.OriginMonth), 1, 0, 0, 0, 0, time.UTC)
			isOverdue := monthDate.Before(twoMonthsAgo)

			groups[key] = &domain.DeferredGroup{
				Year:         tx.OriginYear,
				Month:        tx.OriginMonth,
				MonthLabel:   monthDate.Format("January 2006"),
				Total:        decimal.Zero,
				ItemCount:    0,
				IsOverdue:    isOverdue,
				Transactions: []domain.CCTransactionWithAccount{},
			}
		}

		groups[key].Transactions = append(groups[key].Transactions, *tx)
		groups[key].Total = groups[key].Total.Add(tx.Amount)
		groups[key].ItemCount++
	}

	// Convert to slice and sort by date (oldest first)
	result := make([]domain.DeferredGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Year != result[j].Year {
			return result[i].Year < result[j].Year
		}
		return result[i].Month < result[j].Month
	})

	return result, nil
}

// GetOverdueCCSummary returns a summary of overdue CC transactions for the warning banner
func (s *CCService) GetOverdueCCSummary(workspaceID int32) (*domain.OverdueSummary, error) {
	transactions, err := s.transactionRepo.GetOverdueCC(workspaceID)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return &domain.OverdueSummary{
			HasOverdue:  false,
			TotalAmount: decimal.Zero,
			ItemCount:   0,
			Groups:      []domain.DeferredGroup{},
		}, nil
	}

	// Group by year-month
	groups := make(map[string]*domain.DeferredGroup)
	totalAmount := decimal.Zero

	for _, tx := range transactions {
		key := fmt.Sprintf("%d-%02d", tx.OriginYear, tx.OriginMonth)

		if _, exists := groups[key]; !exists {
			monthDate := time.Date(tx.OriginYear, time.Month(tx.OriginMonth), 1, 0, 0, 0, 0, time.UTC)

			groups[key] = &domain.DeferredGroup{
				Year:         tx.OriginYear,
				Month:        tx.OriginMonth,
				MonthLabel:   monthDate.Format("January 2006"),
				Total:        decimal.Zero,
				ItemCount:    0,
				IsOverdue:    true,
				Transactions: []domain.CCTransactionWithAccount{},
			}
		}

		groups[key].Transactions = append(groups[key].Transactions, *tx)
		groups[key].Total = groups[key].Total.Add(tx.Amount)
		groups[key].ItemCount++
		totalAmount = totalAmount.Add(tx.Amount)
	}

	// Convert to slice and sort by date (oldest first)
	groupList := make([]domain.DeferredGroup, 0, len(groups))
	for _, group := range groups {
		groupList = append(groupList, *group)
	}
	sort.Slice(groupList, func(i, j int) bool {
		if groupList[i].Year != groupList[j].Year {
			return groupList[i].Year < groupList[j].Year
		}
		return groupList[i].Month < groupList[j].Month
	})

	return &domain.OverdueSummary{
		HasOverdue:  true,
		TotalAmount: totalAmount,
		ItemCount:   len(transactions),
		Groups:      groupList,
	}, nil
}

// UpdateOverdueAmount updates the amount of an overdue CC transaction
// This is intentionally simple - no convenience features, only manual edits allowed
func (s *CCService) UpdateOverdueAmount(workspaceID int32, transactionID int32, newAmount decimal.Decimal) (decimal.Decimal, error) {
	// Get the transaction to verify it exists and is overdue
	tx, err := s.transactionRepo.GetByID(workspaceID, transactionID)
	if err != nil {
		return decimal.Zero, domain.ErrTransactionNotFound
	}

	// Verify this is an overdue CC transaction (billed + deferred + 2+ months old)
	if tx.CCState == nil || *tx.CCState != domain.CCStateBilled {
		return decimal.Zero, domain.ErrInvalidCCState
	}

	// Check if billed_at is more than 2 months ago
	if tx.BilledAt == nil {
		return decimal.Zero, domain.ErrInvalidCCState
	}

	twoMonthsAgo := time.Now().AddDate(0, -2, 0)
	if !tx.BilledAt.Before(twoMonthsAgo) {
		return decimal.Zero, domain.ErrInvalidCCState
	}

	// Update only the amount
	err = s.transactionRepo.UpdateAmount(workspaceID, transactionID, newAmount)
	if err != nil {
		return decimal.Zero, err
	}

	return newAmount, nil
}
