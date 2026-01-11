package service

import (
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
