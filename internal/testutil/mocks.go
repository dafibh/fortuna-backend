package testutil

import (
	"fmt"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MockUserRepository is a mock implementation of domain.UserRepository
type MockUserRepository struct {
	Users    map[string]*domain.User
	ByID     map[uuid.UUID]*domain.User
	CreateFn func(auth0ID, email string, name, pictureURL *string) (*domain.User, error)
}

// NewMockUserRepository creates a new MockUserRepository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		Users: make(map[string]*domain.User),
		ByID:  make(map[uuid.UUID]*domain.User),
	}
}

// GetByID retrieves a user by ID
func (m *MockUserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	if user, ok := m.ByID[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

// GetByAuth0ID retrieves a user by Auth0 ID
func (m *MockUserRepository) GetByAuth0ID(auth0ID string) (*domain.User, error) {
	if user, ok := m.Users[auth0ID]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

// Create creates a new user
func (m *MockUserRepository) Create(user *domain.User) (*domain.User, error) {
	user.ID = uuid.New()
	m.Users[user.Auth0ID] = user
	m.ByID[user.ID] = user
	return user, nil
}

// Update updates an existing user
func (m *MockUserRepository) Update(user *domain.User) (*domain.User, error) {
	if _, ok := m.ByID[user.ID]; !ok {
		return nil, domain.ErrUserNotFound
	}
	m.Users[user.Auth0ID] = user
	m.ByID[user.ID] = user
	return user, nil
}

// UpdateName updates only the user's name by Auth0 ID
func (m *MockUserRepository) UpdateName(auth0ID string, name string) (*domain.User, error) {
	user, ok := m.Users[auth0ID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	user.Name = &name
	return user, nil
}

// CreateOrGetByAuth0ID creates or retrieves a user by Auth0 ID
func (m *MockUserRepository) CreateOrGetByAuth0ID(auth0ID, email string, name, pictureURL *string) (*domain.User, error) {
	if m.CreateFn != nil {
		return m.CreateFn(auth0ID, email, name, pictureURL)
	}
	if user, ok := m.Users[auth0ID]; ok {
		return user, nil
	}
	user := &domain.User{
		ID:         uuid.New(),
		Auth0ID:    auth0ID,
		Email:      email,
		Name:       name,
		PictureURL: pictureURL,
	}
	m.Users[auth0ID] = user
	m.ByID[user.ID] = user
	return user, nil
}

// AddUser adds a user to the mock repository (helper for tests)
func (m *MockUserRepository) AddUser(user *domain.User) {
	m.Users[user.Auth0ID] = user
	m.ByID[user.ID] = user
}

// MockWorkspaceRepository is a mock implementation of domain.WorkspaceRepository
type MockWorkspaceRepository struct {
	Workspaces    map[int32]*domain.Workspace
	ByUserID      map[uuid.UUID]*domain.Workspace
	ByUserAuth0ID map[string]*domain.Workspace
	NextID        int32
	GetByUserIDFn func(userID uuid.UUID) (*domain.Workspace, error)
}

// NewMockWorkspaceRepository creates a new MockWorkspaceRepository
func NewMockWorkspaceRepository() *MockWorkspaceRepository {
	return &MockWorkspaceRepository{
		Workspaces:    make(map[int32]*domain.Workspace),
		ByUserID:      make(map[uuid.UUID]*domain.Workspace),
		ByUserAuth0ID: make(map[string]*domain.Workspace),
		NextID:        1,
	}
}

// GetByID retrieves a workspace by ID
func (m *MockWorkspaceRepository) GetByID(id int32) (*domain.Workspace, error) {
	if ws, ok := m.Workspaces[id]; ok {
		return ws, nil
	}
	return nil, domain.ErrWorkspaceNotFound
}

// GetByUserID retrieves a workspace by user ID
func (m *MockWorkspaceRepository) GetByUserID(userID uuid.UUID) (*domain.Workspace, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(userID)
	}
	if ws, ok := m.ByUserID[userID]; ok {
		return ws, nil
	}
	return nil, domain.ErrWorkspaceNotFound
}

// GetByUserAuth0ID retrieves a workspace by user's Auth0 ID
func (m *MockWorkspaceRepository) GetByUserAuth0ID(auth0ID string) (*domain.Workspace, error) {
	if ws, ok := m.ByUserAuth0ID[auth0ID]; ok {
		return ws, nil
	}
	return nil, domain.ErrWorkspaceNotFound
}

// Create creates a new workspace
func (m *MockWorkspaceRepository) Create(workspace *domain.Workspace) (*domain.Workspace, error) {
	workspace.ID = m.NextID
	m.NextID++
	m.Workspaces[workspace.ID] = workspace
	m.ByUserID[workspace.UserID] = workspace
	return workspace, nil
}

// Update updates an existing workspace
func (m *MockWorkspaceRepository) Update(workspace *domain.Workspace) (*domain.Workspace, error) {
	if _, ok := m.Workspaces[workspace.ID]; !ok {
		return nil, domain.ErrWorkspaceNotFound
	}
	m.Workspaces[workspace.ID] = workspace
	m.ByUserID[workspace.UserID] = workspace
	return workspace, nil
}

// Delete deletes a workspace by ID
func (m *MockWorkspaceRepository) Delete(id int32) error {
	ws, ok := m.Workspaces[id]
	if !ok {
		return nil
	}
	delete(m.Workspaces, id)
	delete(m.ByUserID, ws.UserID)
	return nil
}

// AddWorkspace adds a workspace to the mock repository (helper for tests)
func (m *MockWorkspaceRepository) AddWorkspace(workspace *domain.Workspace, auth0ID string) {
	m.Workspaces[workspace.ID] = workspace
	m.ByUserID[workspace.UserID] = workspace
	if auth0ID != "" {
		m.ByUserAuth0ID[auth0ID] = workspace
	}
}

// MockAccountRepository is a mock implementation of domain.AccountRepository
type MockAccountRepository struct {
	Accounts                   map[int32]*domain.Account
	ByWorkspace                map[int32][]*domain.Account
	NextID                     int32
	CreateFn                   func(account *domain.Account) (*domain.Account, error)
	GetByIDFn                  func(workspaceID int32, id int32) (*domain.Account, error)
	GetAllFn                   func(workspaceID int32, includeArchived bool) ([]*domain.Account, error)
	UpdateFn                   func(workspaceID int32, id int32, name string) (*domain.Account, error)
	SoftDeleteFn               func(workspaceID int32, id int32) error
	HardDeleteFn               func(workspaceID int32, id int32) error
	GetCCOutstandingSummaryFn  func(workspaceID int32) (*domain.CCOutstandingSummary, error)
	GetPerAccountOutstandingFn func(workspaceID int32) ([]*domain.PerAccountOutstanding, error)
}

// NewMockAccountRepository creates a new MockAccountRepository
func NewMockAccountRepository() *MockAccountRepository {
	return &MockAccountRepository{
		Accounts:    make(map[int32]*domain.Account),
		ByWorkspace: make(map[int32][]*domain.Account),
		NextID:      1,
	}
}

// Create creates a new account
func (m *MockAccountRepository) Create(account *domain.Account) (*domain.Account, error) {
	if m.CreateFn != nil {
		return m.CreateFn(account)
	}
	account.ID = m.NextID
	m.NextID++
	m.Accounts[account.ID] = account
	m.ByWorkspace[account.WorkspaceID] = append(m.ByWorkspace[account.WorkspaceID], account)
	return account, nil
}

// GetByID retrieves an account by its ID within a workspace
func (m *MockAccountRepository) GetByID(workspaceID int32, id int32) (*domain.Account, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	account, ok := m.Accounts[id]
	if !ok || account.WorkspaceID != workspaceID {
		return nil, domain.ErrAccountNotFound
	}
	// Check if soft-deleted
	if account.DeletedAt != nil {
		return nil, domain.ErrAccountNotFound
	}
	return account, nil
}

// GetAllByWorkspace retrieves all accounts for a workspace
func (m *MockAccountRepository) GetAllByWorkspace(workspaceID int32, includeArchived bool) ([]*domain.Account, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(workspaceID, includeArchived)
	}
	accounts := m.ByWorkspace[workspaceID]
	if accounts == nil {
		return []*domain.Account{}, nil
	}
	if includeArchived {
		return accounts, nil
	}
	// Filter out soft-deleted accounts
	var active []*domain.Account
	for _, acc := range accounts {
		if acc.DeletedAt == nil {
			active = append(active, acc)
		}
	}
	if active == nil {
		return []*domain.Account{}, nil
	}
	return active, nil
}

// Update updates an account's name
func (m *MockAccountRepository) Update(workspaceID int32, id int32, name string) (*domain.Account, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(workspaceID, id, name)
	}
	account, ok := m.Accounts[id]
	if !ok || account.WorkspaceID != workspaceID || account.DeletedAt != nil {
		return nil, domain.ErrAccountNotFound
	}
	account.Name = name
	return account, nil
}

// SoftDelete marks an account as deleted
func (m *MockAccountRepository) SoftDelete(workspaceID int32, id int32) error {
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(workspaceID, id)
	}
	account, ok := m.Accounts[id]
	if !ok || account.WorkspaceID != workspaceID || account.DeletedAt != nil {
		return domain.ErrAccountNotFound
	}
	now := account.UpdatedAt
	account.DeletedAt = &now
	return nil
}

// HardDelete permanently removes an account
func (m *MockAccountRepository) HardDelete(workspaceID int32, id int32) error {
	if m.HardDeleteFn != nil {
		return m.HardDeleteFn(workspaceID, id)
	}
	account, ok := m.Accounts[id]
	if !ok || account.WorkspaceID != workspaceID {
		return nil
	}
	delete(m.Accounts, id)
	// Remove from ByWorkspace slice
	accounts := m.ByWorkspace[workspaceID]
	for i, acc := range accounts {
		if acc.ID == id {
			m.ByWorkspace[workspaceID] = append(accounts[:i], accounts[i+1:]...)
			break
		}
	}
	return nil
}

// AddAccount adds an account to the mock repository (helper for tests)
func (m *MockAccountRepository) AddAccount(account *domain.Account) {
	m.Accounts[account.ID] = account
	m.ByWorkspace[account.WorkspaceID] = append(m.ByWorkspace[account.WorkspaceID], account)
}

// GetCCOutstandingSummary returns total CC outstanding across all CC accounts
func (m *MockAccountRepository) GetCCOutstandingSummary(workspaceID int32) (*domain.CCOutstandingSummary, error) {
	if m.GetCCOutstandingSummaryFn != nil {
		return m.GetCCOutstandingSummaryFn(workspaceID)
	}
	// Default: return zeros
	return &domain.CCOutstandingSummary{
		TotalOutstanding: decimal.Zero,
		CCAccountCount:   0,
	}, nil
}

// GetPerAccountOutstanding returns outstanding balance for each CC account
func (m *MockAccountRepository) GetPerAccountOutstanding(workspaceID int32) ([]*domain.PerAccountOutstanding, error) {
	if m.GetPerAccountOutstandingFn != nil {
		return m.GetPerAccountOutstandingFn(workspaceID)
	}
	// Default: return empty list
	return []*domain.PerAccountOutstanding{}, nil
}

// MockTransactionRepository is a mock implementation of domain.TransactionRepository
type MockTransactionRepository struct {
	Transactions               map[int32]*domain.Transaction
	ByWorkspace                map[int32][]*domain.Transaction
	ByTransferPairID           map[uuid.UUID][]*domain.Transaction
	NextID                     int32
	CreateFn                   func(transaction *domain.Transaction) (*domain.Transaction, error)
	GetByIDFn                  func(workspaceID int32, id int32) (*domain.Transaction, error)
	GetByWSFn                  func(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error)
	TogglePaidFn               func(workspaceID int32, id int32) (*domain.Transaction, error)
	UpdateSettlementIntentFn   func(workspaceID int32, id int32, intent domain.CCSettlementIntent) (*domain.Transaction, error)
	UpdateFn                   func(workspaceID int32, id int32, data *domain.UpdateTransactionData) (*domain.Transaction, error)
	SoftDeleteFn                      func(workspaceID int32, id int32) error
	CreateTransferPairFn              func(fromTx, toTx *domain.Transaction) (*domain.TransferResult, error)
	SoftDeleteTransferPairFn          func(workspaceID int32, pairID uuid.UUID) error
	GetAccountTransactionSummariesFn  func(workspaceID int32) ([]*domain.TransactionSummary, error)
	SumByTypeAndDateRangeFn           func(workspaceID int32, startDate, endDate time.Time, txType domain.TransactionType) (decimal.Decimal, error)
	SumPaidExpensesByDateRangeFn      func(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	SumUnpaidExpensesByDateRangeFn    func(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error)
	GetCCPayableSummaryFn             func(workspaceID int32) ([]*domain.CCPayableSummaryRow, error)
	GetRecentlyUsedCategoriesFn       func(workspaceID int32) ([]*domain.RecentCategory, error)
	GetCCPayableBreakdownFn           func(workspaceID int32) ([]*domain.CCPayableTransaction, error)
}

// NewMockTransactionRepository creates a new MockTransactionRepository
func NewMockTransactionRepository() *MockTransactionRepository {
	return &MockTransactionRepository{
		Transactions:     make(map[int32]*domain.Transaction),
		ByWorkspace:      make(map[int32][]*domain.Transaction),
		ByTransferPairID: make(map[uuid.UUID][]*domain.Transaction),
		NextID:           1,
	}
}

// Create creates a new transaction
func (m *MockTransactionRepository) Create(transaction *domain.Transaction) (*domain.Transaction, error) {
	if m.CreateFn != nil {
		return m.CreateFn(transaction)
	}
	transaction.ID = m.NextID
	m.NextID++
	m.Transactions[transaction.ID] = transaction
	m.ByWorkspace[transaction.WorkspaceID] = append(m.ByWorkspace[transaction.WorkspaceID], transaction)
	return transaction, nil
}

// GetByID retrieves a transaction by its ID within a workspace
func (m *MockTransactionRepository) GetByID(workspaceID int32, id int32) (*domain.Transaction, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	transaction, ok := m.Transactions[id]
	if !ok || transaction.WorkspaceID != workspaceID {
		return nil, domain.ErrTransactionNotFound
	}
	if transaction.DeletedAt != nil {
		return nil, domain.ErrTransactionNotFound
	}
	return transaction, nil
}

// GetByWorkspace retrieves all transactions for a workspace with optional filters and pagination
func (m *MockTransactionRepository) GetByWorkspace(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error) {
	if m.GetByWSFn != nil {
		return m.GetByWSFn(workspaceID, filters)
	}
	transactions := m.ByWorkspace[workspaceID]
	if transactions == nil {
		transactions = []*domain.Transaction{}
	}

	// Filter out soft-deleted and apply filters
	var filtered []*domain.Transaction
	for _, t := range transactions {
		if t.DeletedAt != nil {
			continue
		}
		if filters != nil {
			if filters.AccountID != nil && t.AccountID != *filters.AccountID {
				continue
			}
			if filters.StartDate != nil && t.TransactionDate.Before(*filters.StartDate) {
				continue
			}
			if filters.EndDate != nil && t.TransactionDate.After(*filters.EndDate) {
				continue
			}
			if filters.Type != nil && t.Type != *filters.Type {
				continue
			}
		}
		filtered = append(filtered, t)
	}
	if filtered == nil {
		filtered = []*domain.Transaction{}
	}

	// Apply pagination
	page := int32(1)
	pageSize := int32(domain.DefaultPageSize)
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	totalItems := int64(len(filtered))
	totalPages := int32(totalItems / int64(pageSize))
	if totalItems%int64(pageSize) > 0 {
		totalPages++
	}

	// Apply offset and limit
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= int32(len(filtered)) {
		filtered = []*domain.Transaction{}
	} else {
		if end > int32(len(filtered)) {
			end = int32(len(filtered))
		}
		filtered = filtered[start:end]
	}

	return &domain.PaginatedTransactions{
		Data:       filtered,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

// TogglePaid toggles the paid status of a transaction
func (m *MockTransactionRepository) TogglePaid(workspaceID int32, id int32) (*domain.Transaction, error) {
	if m.TogglePaidFn != nil {
		return m.TogglePaidFn(workspaceID, id)
	}
	transaction, ok := m.Transactions[id]
	if !ok || transaction.WorkspaceID != workspaceID {
		return nil, domain.ErrTransactionNotFound
	}
	if transaction.DeletedAt != nil {
		return nil, domain.ErrTransactionNotFound
	}
	transaction.IsPaid = !transaction.IsPaid
	return transaction, nil
}

// UpdateSettlementIntent updates the CC settlement intent for an unpaid transaction
func (m *MockTransactionRepository) UpdateSettlementIntent(workspaceID int32, id int32, intent domain.CCSettlementIntent) (*domain.Transaction, error) {
	if m.UpdateSettlementIntentFn != nil {
		return m.UpdateSettlementIntentFn(workspaceID, id, intent)
	}
	transaction, ok := m.Transactions[id]
	if !ok || transaction.WorkspaceID != workspaceID {
		return nil, domain.ErrTransactionNotFound
	}
	if transaction.DeletedAt != nil {
		return nil, domain.ErrTransactionNotFound
	}
	// Simulate the SQL constraint: is_paid = false
	if transaction.IsPaid {
		return nil, domain.ErrTransactionNotFound
	}
	transaction.CCSettlementIntent = &intent
	return transaction, nil
}

// Update updates a transaction
func (m *MockTransactionRepository) Update(workspaceID int32, id int32, data *domain.UpdateTransactionData) (*domain.Transaction, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(workspaceID, id, data)
	}
	transaction, ok := m.Transactions[id]
	if !ok || transaction.WorkspaceID != workspaceID {
		return nil, domain.ErrTransactionNotFound
	}
	if transaction.DeletedAt != nil {
		return nil, domain.ErrTransactionNotFound
	}
	transaction.Name = data.Name
	transaction.Amount = data.Amount
	transaction.Type = data.Type
	transaction.TransactionDate = data.TransactionDate
	transaction.AccountID = data.AccountID
	transaction.CCSettlementIntent = data.CCSettlementIntent
	transaction.Notes = data.Notes
	transaction.CategoryID = data.CategoryID
	return transaction, nil
}

// SoftDelete soft deletes a transaction
func (m *MockTransactionRepository) SoftDelete(workspaceID int32, id int32) error {
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(workspaceID, id)
	}
	transaction, ok := m.Transactions[id]
	if !ok || transaction.WorkspaceID != workspaceID {
		return domain.ErrTransactionNotFound
	}
	if transaction.DeletedAt != nil {
		return domain.ErrTransactionNotFound
	}
	now := time.Now()
	transaction.DeletedAt = &now
	return nil
}

// AddTransaction adds a transaction to the mock repository (helper for tests)
func (m *MockTransactionRepository) AddTransaction(transaction *domain.Transaction) {
	m.Transactions[transaction.ID] = transaction
	m.ByWorkspace[transaction.WorkspaceID] = append(m.ByWorkspace[transaction.WorkspaceID], transaction)
	if transaction.TransferPairID != nil {
		m.ByTransferPairID[*transaction.TransferPairID] = append(m.ByTransferPairID[*transaction.TransferPairID], transaction)
	}
}

// CreateTransferPair creates two linked transactions atomically
func (m *MockTransactionRepository) CreateTransferPair(fromTx, toTx *domain.Transaction) (*domain.TransferResult, error) {
	if m.CreateTransferPairFn != nil {
		return m.CreateTransferPairFn(fromTx, toTx)
	}
	// Assign IDs
	fromTx.ID = m.NextID
	m.NextID++
	toTx.ID = m.NextID
	m.NextID++

	// Store transactions
	m.Transactions[fromTx.ID] = fromTx
	m.Transactions[toTx.ID] = toTx
	m.ByWorkspace[fromTx.WorkspaceID] = append(m.ByWorkspace[fromTx.WorkspaceID], fromTx)
	m.ByWorkspace[toTx.WorkspaceID] = append(m.ByWorkspace[toTx.WorkspaceID], toTx)
	if fromTx.TransferPairID != nil {
		m.ByTransferPairID[*fromTx.TransferPairID] = append(m.ByTransferPairID[*fromTx.TransferPairID], fromTx, toTx)
	}

	return &domain.TransferResult{
		FromTransaction: fromTx,
		ToTransaction:   toTx,
	}, nil
}

// SoftDeleteTransferPair soft deletes both transactions in a transfer pair
func (m *MockTransactionRepository) SoftDeleteTransferPair(workspaceID int32, pairID uuid.UUID) error {
	if m.SoftDeleteTransferPairFn != nil {
		return m.SoftDeleteTransferPairFn(workspaceID, pairID)
	}
	transactions, ok := m.ByTransferPairID[pairID]
	if !ok || len(transactions) == 0 {
		return domain.ErrTransactionNotFound
	}
	now := time.Now()
	for _, tx := range transactions {
		if tx.WorkspaceID == workspaceID && tx.DeletedAt == nil {
			tx.DeletedAt = &now
		}
	}
	return nil
}

// GetAccountTransactionSummaries returns aggregated transaction data for balance calculations
func (m *MockTransactionRepository) GetAccountTransactionSummaries(workspaceID int32) ([]*domain.TransactionSummary, error) {
	if m.GetAccountTransactionSummariesFn != nil {
		return m.GetAccountTransactionSummariesFn(workspaceID)
	}

	// Aggregate transactions by account
	summaryMap := make(map[int32]*domain.TransactionSummary)
	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		summary, ok := summaryMap[tx.AccountID]
		if !ok {
			summary = &domain.TransactionSummary{AccountID: tx.AccountID}
			summaryMap[tx.AccountID] = summary
		}
		if tx.Type == domain.TransactionTypeIncome {
			summary.SumIncome = summary.SumIncome.Add(tx.Amount)
		} else if tx.Type == domain.TransactionTypeExpense {
			summary.SumExpenses = summary.SumExpenses.Add(tx.Amount)
			if !tx.IsPaid {
				summary.SumUnpaidExpenses = summary.SumUnpaidExpenses.Add(tx.Amount)
			}
		}
	}

	summaries := make([]*domain.TransactionSummary, 0, len(summaryMap))
	for _, s := range summaryMap {
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// SumByTypeAndDateRange sums transactions by type within a date range
func (m *MockTransactionRepository) SumByTypeAndDateRange(workspaceID int32, startDate, endDate time.Time, txType domain.TransactionType) (decimal.Decimal, error) {
	if m.SumByTypeAndDateRangeFn != nil {
		return m.SumByTypeAndDateRangeFn(workspaceID, startDate, endDate, txType)
	}

	total := decimal.Zero
	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		if tx.Type != txType {
			continue
		}
		// Check if transaction date is within range (inclusive)
		if (tx.TransactionDate.Equal(startDate) || tx.TransactionDate.After(startDate)) &&
			(tx.TransactionDate.Equal(endDate) || tx.TransactionDate.Before(endDate)) {
			total = total.Add(tx.Amount)
		}
	}
	return total, nil
}

// GetMonthlyTransactionSummaries returns income/expense totals grouped by year/month
func (m *MockTransactionRepository) GetMonthlyTransactionSummaries(workspaceID int32) ([]*domain.MonthlyTransactionSummary, error) {
	// Aggregate transactions by year/month
	type monthKey struct {
		year  int
		month int
	}
	summaryMap := make(map[monthKey]*domain.MonthlyTransactionSummary)

	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		key := monthKey{year: tx.TransactionDate.Year(), month: int(tx.TransactionDate.Month())}
		summary, ok := summaryMap[key]
		if !ok {
			summary = &domain.MonthlyTransactionSummary{Year: key.year, Month: key.month}
			summaryMap[key] = summary
		}
		if tx.Type == domain.TransactionTypeIncome {
			summary.TotalIncome = summary.TotalIncome.Add(tx.Amount)
		} else if tx.Type == domain.TransactionTypeExpense {
			summary.TotalExpenses = summary.TotalExpenses.Add(tx.Amount)
		}
	}

	summaries := make([]*domain.MonthlyTransactionSummary, 0, len(summaryMap))
	for _, s := range summaryMap {
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// SumPaidExpensesByDateRange sums paid expenses within a date range
func (m *MockTransactionRepository) SumPaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error) {
	if m.SumPaidExpensesByDateRangeFn != nil {
		return m.SumPaidExpensesByDateRangeFn(workspaceID, startDate, endDate)
	}

	total := decimal.Zero
	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		if tx.Type != domain.TransactionTypeExpense {
			continue
		}
		if !tx.IsPaid {
			continue
		}
		// Check if transaction date is within range (inclusive)
		if (tx.TransactionDate.Equal(startDate) || tx.TransactionDate.After(startDate)) &&
			(tx.TransactionDate.Equal(endDate) || tx.TransactionDate.Before(endDate)) {
			total = total.Add(tx.Amount)
		}
	}
	return total, nil
}

// SumUnpaidExpensesByDateRange sums unpaid expenses within a date range
func (m *MockTransactionRepository) SumUnpaidExpensesByDateRange(workspaceID int32, startDate, endDate time.Time) (decimal.Decimal, error) {
	if m.SumUnpaidExpensesByDateRangeFn != nil {
		return m.SumUnpaidExpensesByDateRangeFn(workspaceID, startDate, endDate)
	}

	total := decimal.Zero
	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		if tx.Type != domain.TransactionTypeExpense {
			continue
		}
		if tx.IsPaid {
			continue
		}
		// Check if transaction date is within range (inclusive)
		if (tx.TransactionDate.Equal(startDate) || tx.TransactionDate.After(startDate)) &&
			(tx.TransactionDate.Equal(endDate) || tx.TransactionDate.Before(endDate)) {
			total = total.Add(tx.Amount)
		}
	}
	return total, nil
}

// GetCCPayableSummary returns unpaid CC transaction totals grouped by settlement intent
// NOTE: This mock does NOT verify account template like the real SQL query does.
// The real query joins with accounts table to filter by template='credit_card'.
// Tests using this mock should be aware of this limitation - use GetCCPayableSummaryFn
// for tests requiring precise control over the query behavior.
func (m *MockTransactionRepository) GetCCPayableSummary(workspaceID int32) ([]*domain.CCPayableSummaryRow, error) {
	if m.GetCCPayableSummaryFn != nil {
		return m.GetCCPayableSummaryFn(workspaceID)
	}

	// Note: The real implementation joins with accounts to filter by CC template.
	// In the mock, we don't have access to accounts, so we just aggregate by settlement intent
	// for transactions that have a settlement intent set (which implies CC).
	summaryMap := make(map[domain.CCSettlementIntent]decimal.Decimal)

	for _, tx := range m.ByWorkspace[workspaceID] {
		if tx.DeletedAt != nil {
			continue
		}
		if tx.Type != domain.TransactionTypeExpense {
			continue
		}
		if tx.IsPaid {
			continue
		}
		if tx.CCSettlementIntent == nil {
			continue
		}
		intent := *tx.CCSettlementIntent
		summaryMap[intent] = summaryMap[intent].Add(tx.Amount)
	}

	var result []*domain.CCPayableSummaryRow
	for intent, total := range summaryMap {
		result = append(result, &domain.CCPayableSummaryRow{
			SettlementIntent: intent,
			Total:            total,
		})
	}
	return result, nil
}

// GetRecentlyUsedCategories returns recently used categories for suggestions
func (m *MockTransactionRepository) GetRecentlyUsedCategories(workspaceID int32) ([]*domain.RecentCategory, error) {
	if m.GetRecentlyUsedCategoriesFn != nil {
		return m.GetRecentlyUsedCategoriesFn(workspaceID)
	}
	// Default: return empty list
	return []*domain.RecentCategory{}, nil
}

// GetCCPayableBreakdown returns CC transactions for payable breakdown
func (m *MockTransactionRepository) GetCCPayableBreakdown(workspaceID int32) ([]*domain.CCPayableTransaction, error) {
	if m.GetCCPayableBreakdownFn != nil {
		return m.GetCCPayableBreakdownFn(workspaceID)
	}
	// Default: return empty list
	return []*domain.CCPayableTransaction{}, nil
}

// MockMonthRepository is a mock implementation of domain.MonthRepository
type MockMonthRepository struct {
	Months                             map[int32]*domain.Month
	ByWorkspaceYearMonth               map[string]*domain.Month
	NextID                             int32
	CreateFn                           func(workspaceID int32, year, month int, startDate, endDate time.Time, startingBalance decimal.Decimal) (*domain.Month, error)
	GetByYearMonthFn                   func(workspaceID int32, year, month int) (*domain.Month, error)
	GetLatestFn                        func(workspaceID int32) (*domain.Month, error)
	GetAllFn                           func(workspaceID int32) ([]*domain.Month, error)
	UpdateStartingBalanceFn            func(workspaceID, id int32, balance decimal.Decimal) error
}

// NewMockMonthRepository creates a new MockMonthRepository
func NewMockMonthRepository() *MockMonthRepository {
	return &MockMonthRepository{
		Months:               make(map[int32]*domain.Month),
		ByWorkspaceYearMonth: make(map[string]*domain.Month),
		NextID:               1,
	}
}

func monthKey(workspaceID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d", workspaceID, year, month)
}

// Create creates a new month
func (m *MockMonthRepository) Create(workspaceID int32, year, month int, startDate, endDate time.Time, startingBalance decimal.Decimal) (*domain.Month, error) {
	if m.CreateFn != nil {
		return m.CreateFn(workspaceID, year, month, startDate, endDate, startingBalance)
	}
	key := monthKey(workspaceID, year, month)
	if _, exists := m.ByWorkspaceYearMonth[key]; exists {
		return nil, domain.ErrMonthAlreadyExists
	}
	newMonth := &domain.Month{
		ID:              m.NextID,
		WorkspaceID:     workspaceID,
		Year:            year,
		Month:           month,
		StartDate:       startDate,
		EndDate:         endDate,
		StartingBalance: startingBalance,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	m.NextID++
	m.Months[newMonth.ID] = newMonth
	m.ByWorkspaceYearMonth[key] = newMonth
	return newMonth, nil
}

// GetByYearMonth retrieves a month by year and month number
func (m *MockMonthRepository) GetByYearMonth(workspaceID int32, year, month int) (*domain.Month, error) {
	if m.GetByYearMonthFn != nil {
		return m.GetByYearMonthFn(workspaceID, year, month)
	}
	key := monthKey(workspaceID, year, month)
	if mon, ok := m.ByWorkspaceYearMonth[key]; ok {
		return mon, nil
	}
	return nil, domain.ErrMonthNotFound
}

// GetLatest retrieves the most recent month for a workspace
func (m *MockMonthRepository) GetLatest(workspaceID int32) (*domain.Month, error) {
	if m.GetLatestFn != nil {
		return m.GetLatestFn(workspaceID)
	}
	var latest *domain.Month
	for _, mon := range m.Months {
		if mon.WorkspaceID != workspaceID {
			continue
		}
		if latest == nil || (mon.Year > latest.Year) || (mon.Year == latest.Year && mon.Month > latest.Month) {
			latest = mon
		}
	}
	if latest == nil {
		return nil, domain.ErrMonthNotFound
	}
	return latest, nil
}

// GetAll retrieves all months for a workspace
func (m *MockMonthRepository) GetAll(workspaceID int32) ([]*domain.Month, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(workspaceID)
	}
	var result []*domain.Month
	for _, mon := range m.Months {
		if mon.WorkspaceID == workspaceID {
			result = append(result, mon)
		}
	}
	return result, nil
}

// UpdateStartingBalance updates the starting balance of a month
func (m *MockMonthRepository) UpdateStartingBalance(workspaceID, id int32, balance decimal.Decimal) error {
	if m.UpdateStartingBalanceFn != nil {
		return m.UpdateStartingBalanceFn(workspaceID, id, balance)
	}
	mon, ok := m.Months[id]
	if !ok || mon.WorkspaceID != workspaceID {
		return domain.ErrMonthNotFound
	}
	mon.StartingBalance = balance
	mon.UpdatedAt = time.Now()
	return nil
}

// AddMonth adds a month to the mock repository (helper for tests)
func (m *MockMonthRepository) AddMonth(month *domain.Month) {
	m.Months[month.ID] = month
	key := monthKey(month.WorkspaceID, month.Year, month.Month)
	m.ByWorkspaceYearMonth[key] = month
}

// MockBudgetCategoryRepository is a mock implementation of domain.BudgetCategoryRepository
type MockBudgetCategoryRepository struct {
	Categories       map[int32]*domain.BudgetCategory
	ByWorkspace      map[int32][]*domain.BudgetCategory
	ByName           map[string]*domain.BudgetCategory
	NextID           int32
	CreateFn         func(category *domain.BudgetCategory) (*domain.BudgetCategory, error)
	GetByIDFn        func(workspaceID int32, id int32) (*domain.BudgetCategory, error)
	GetByNameFn      func(workspaceID int32, name string) (*domain.BudgetCategory, error)
	GetAllFn         func(workspaceID int32) ([]*domain.BudgetCategory, error)
	UpdateFn         func(workspaceID int32, id int32, name string) (*domain.BudgetCategory, error)
	SoftDeleteFn     func(workspaceID int32, id int32) error
	HasTransactionsFn func(workspaceID int32, id int32) (bool, error)
}

// NewMockBudgetCategoryRepository creates a new MockBudgetCategoryRepository
func NewMockBudgetCategoryRepository() *MockBudgetCategoryRepository {
	return &MockBudgetCategoryRepository{
		Categories:  make(map[int32]*domain.BudgetCategory),
		ByWorkspace: make(map[int32][]*domain.BudgetCategory),
		ByName:      make(map[string]*domain.BudgetCategory),
		NextID:      1,
	}
}

func budgetCategoryNameKey(workspaceID int32, name string) string {
	return fmt.Sprintf("%d-%s", workspaceID, name)
}

// Create creates a new budget category
func (m *MockBudgetCategoryRepository) Create(category *domain.BudgetCategory) (*domain.BudgetCategory, error) {
	if m.CreateFn != nil {
		return m.CreateFn(category)
	}
	// Check for duplicate name
	key := budgetCategoryNameKey(category.WorkspaceID, category.Name)
	if existing, ok := m.ByName[key]; ok && existing.DeletedAt == nil {
		return nil, domain.ErrBudgetCategoryAlreadyExists
	}
	category.ID = m.NextID
	m.NextID++
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()
	m.Categories[category.ID] = category
	m.ByWorkspace[category.WorkspaceID] = append(m.ByWorkspace[category.WorkspaceID], category)
	m.ByName[key] = category
	return category, nil
}

// GetByID retrieves a budget category by its ID within a workspace
func (m *MockBudgetCategoryRepository) GetByID(workspaceID int32, id int32) (*domain.BudgetCategory, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	category, ok := m.Categories[id]
	if !ok || category.WorkspaceID != workspaceID {
		return nil, domain.ErrBudgetCategoryNotFound
	}
	if category.DeletedAt != nil {
		return nil, domain.ErrBudgetCategoryNotFound
	}
	return category, nil
}

// GetByName retrieves a budget category by its name within a workspace
func (m *MockBudgetCategoryRepository) GetByName(workspaceID int32, name string) (*domain.BudgetCategory, error) {
	if m.GetByNameFn != nil {
		return m.GetByNameFn(workspaceID, name)
	}
	key := budgetCategoryNameKey(workspaceID, name)
	category, ok := m.ByName[key]
	if !ok || category.DeletedAt != nil {
		return nil, domain.ErrBudgetCategoryNotFound
	}
	return category, nil
}

// GetAllByWorkspace retrieves all budget categories for a workspace
func (m *MockBudgetCategoryRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.BudgetCategory, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(workspaceID)
	}
	categories := m.ByWorkspace[workspaceID]
	if categories == nil {
		return []*domain.BudgetCategory{}, nil
	}
	// Filter out soft-deleted categories
	var active []*domain.BudgetCategory
	for _, cat := range categories {
		if cat.DeletedAt == nil {
			active = append(active, cat)
		}
	}
	if active == nil {
		return []*domain.BudgetCategory{}, nil
	}
	return active, nil
}

// Update updates a budget category's name
func (m *MockBudgetCategoryRepository) Update(workspaceID int32, id int32, name string) (*domain.BudgetCategory, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(workspaceID, id, name)
	}
	category, ok := m.Categories[id]
	if !ok || category.WorkspaceID != workspaceID || category.DeletedAt != nil {
		return nil, domain.ErrBudgetCategoryNotFound
	}
	// Check for duplicate name (excluding self)
	key := budgetCategoryNameKey(workspaceID, name)
	if existing, ok := m.ByName[key]; ok && existing.ID != id && existing.DeletedAt == nil {
		return nil, domain.ErrBudgetCategoryAlreadyExists
	}
	// Remove old name key
	oldKey := budgetCategoryNameKey(workspaceID, category.Name)
	delete(m.ByName, oldKey)
	// Update
	category.Name = name
	category.UpdatedAt = time.Now()
	m.ByName[key] = category
	return category, nil
}

// SoftDelete marks a budget category as deleted
func (m *MockBudgetCategoryRepository) SoftDelete(workspaceID int32, id int32) error {
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(workspaceID, id)
	}
	category, ok := m.Categories[id]
	if !ok || category.WorkspaceID != workspaceID || category.DeletedAt != nil {
		return domain.ErrBudgetCategoryNotFound
	}
	now := time.Now()
	category.DeletedAt = &now
	return nil
}

// HasTransactions checks if a budget category has any transactions assigned
func (m *MockBudgetCategoryRepository) HasTransactions(workspaceID int32, id int32) (bool, error) {
	if m.HasTransactionsFn != nil {
		return m.HasTransactionsFn(workspaceID, id)
	}
	// Default: no transactions (until Story 4.2)
	return false, nil
}

// AddBudgetCategory adds a budget category to the mock repository (helper for tests)
func (m *MockBudgetCategoryRepository) AddBudgetCategory(category *domain.BudgetCategory) {
	m.Categories[category.ID] = category
	m.ByWorkspace[category.WorkspaceID] = append(m.ByWorkspace[category.WorkspaceID], category)
	key := budgetCategoryNameKey(category.WorkspaceID, category.Name)
	m.ByName[key] = category
}

// MockBudgetAllocationRepository is a mock implementation of domain.BudgetAllocationRepository
type MockBudgetAllocationRepository struct {
	Allocations               map[string]*domain.BudgetAllocation
	ByWorkspaceMonth          map[string][]*domain.BudgetAllocation
	CategoriesWithAllocations map[string][]*domain.BudgetCategoryWithAllocation
	SpendingByCategory        map[string][]*domain.CategorySpending
	AllocationCounts          map[string]int64
	NextID                    int32
	UpsertFn                  func(allocation *domain.BudgetAllocation) (*domain.BudgetAllocation, error)
	UpsertBatchFn             func(allocations []*domain.BudgetAllocation) error
	GetByMonthFn              func(workspaceID int32, year, month int) ([]*domain.BudgetAllocation, error)
	GetByCategoryFn           func(workspaceID int32, categoryID int32, year, month int) (*domain.BudgetAllocation, error)
	DeleteFn                  func(workspaceID int32, categoryID int32, year, month int) error
	GetCategoriesWithAllocationsFn func(workspaceID int32, year, month int) ([]*domain.BudgetCategoryWithAllocation, error)
	GetSpendingByCategoryFn        func(workspaceID int32, year, month int) ([]*domain.CategorySpending, error)
	GetCategoryTransactionsFn      func(workspaceID int32, categoryID int32, year, month int) ([]*domain.CategoryTransaction, error)
	CountAllocationsForMonthFn     func(workspaceID int32, year, month int) (int64, error)
	CopyAllocationsToMonthFn       func(workspaceID int32, fromYear, fromMonth, toYear, toMonth int) error
}

// NewMockBudgetAllocationRepository creates a new MockBudgetAllocationRepository
func NewMockBudgetAllocationRepository() *MockBudgetAllocationRepository {
	return &MockBudgetAllocationRepository{
		Allocations:               make(map[string]*domain.BudgetAllocation),
		ByWorkspaceMonth:          make(map[string][]*domain.BudgetAllocation),
		CategoriesWithAllocations: make(map[string][]*domain.BudgetCategoryWithAllocation),
		SpendingByCategory:        make(map[string][]*domain.CategorySpending),
		AllocationCounts:          make(map[string]int64),
		NextID:                    1,
	}
}

func allocationKey(workspaceID, categoryID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d-%d", workspaceID, categoryID, year, month)
}

func allocationMonthKey(workspaceID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d", workspaceID, year, month)
}

// Upsert creates or updates a budget allocation
func (m *MockBudgetAllocationRepository) Upsert(allocation *domain.BudgetAllocation) (*domain.BudgetAllocation, error) {
	if m.UpsertFn != nil {
		return m.UpsertFn(allocation)
	}
	key := allocationKey(allocation.WorkspaceID, allocation.CategoryID, allocation.Year, allocation.Month)
	monthKey := allocationMonthKey(allocation.WorkspaceID, allocation.Year, allocation.Month)

	existing, exists := m.Allocations[key]
	if exists {
		// Update existing
		existing.Amount = allocation.Amount
		existing.UpdatedAt = time.Now()
		return existing, nil
	}

	// Create new
	allocation.ID = m.NextID
	m.NextID++
	allocation.CreatedAt = time.Now()
	allocation.UpdatedAt = time.Now()
	m.Allocations[key] = allocation
	m.ByWorkspaceMonth[monthKey] = append(m.ByWorkspaceMonth[monthKey], allocation)
	return allocation, nil
}

// UpsertBatch creates or updates multiple budget allocations atomically
func (m *MockBudgetAllocationRepository) UpsertBatch(allocations []*domain.BudgetAllocation) error {
	if m.UpsertBatchFn != nil {
		return m.UpsertBatchFn(allocations)
	}
	for _, allocation := range allocations {
		_, err := m.Upsert(allocation)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetByMonth retrieves all budget allocations for a specific month
func (m *MockBudgetAllocationRepository) GetByMonth(workspaceID int32, year, month int) ([]*domain.BudgetAllocation, error) {
	if m.GetByMonthFn != nil {
		return m.GetByMonthFn(workspaceID, year, month)
	}
	key := allocationMonthKey(workspaceID, year, month)
	allocations := m.ByWorkspaceMonth[key]
	if allocations == nil {
		return []*domain.BudgetAllocation{}, nil
	}
	return allocations, nil
}

// GetByCategory retrieves a budget allocation for a specific category and month
func (m *MockBudgetAllocationRepository) GetByCategory(workspaceID int32, categoryID int32, year, month int) (*domain.BudgetAllocation, error) {
	if m.GetByCategoryFn != nil {
		return m.GetByCategoryFn(workspaceID, categoryID, year, month)
	}
	key := allocationKey(workspaceID, categoryID, year, month)
	allocation, ok := m.Allocations[key]
	if !ok {
		return nil, domain.ErrBudgetAllocationNotFound
	}
	return allocation, nil
}

// Delete removes a budget allocation
func (m *MockBudgetAllocationRepository) Delete(workspaceID int32, categoryID int32, year, month int) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(workspaceID, categoryID, year, month)
	}
	key := allocationKey(workspaceID, categoryID, year, month)
	delete(m.Allocations, key)

	// Remove from month list
	monthKey := allocationMonthKey(workspaceID, year, month)
	allocations := m.ByWorkspaceMonth[monthKey]
	for i, a := range allocations {
		if a.CategoryID == categoryID {
			m.ByWorkspaceMonth[monthKey] = append(allocations[:i], allocations[i+1:]...)
			break
		}
	}
	return nil
}

// GetCategoriesWithAllocations retrieves all categories with their allocations for a month
func (m *MockBudgetAllocationRepository) GetCategoriesWithAllocations(workspaceID int32, year, month int) ([]*domain.BudgetCategoryWithAllocation, error) {
	if m.GetCategoriesWithAllocationsFn != nil {
		return m.GetCategoriesWithAllocationsFn(workspaceID, year, month)
	}
	key := allocationMonthKey(workspaceID, year, month)
	categories := m.CategoriesWithAllocations[key]
	if categories == nil {
		return []*domain.BudgetCategoryWithAllocation{}, nil
	}
	return categories, nil
}

// SetCategoriesWithAllocations sets the categories with allocations for a month (helper for tests)
func (m *MockBudgetAllocationRepository) SetCategoriesWithAllocations(workspaceID int32, year, month int, categories []*domain.BudgetCategoryWithAllocation) {
	key := allocationMonthKey(workspaceID, year, month)
	m.CategoriesWithAllocations[key] = categories
}

// GetSpendingByCategory retrieves spending totals by category for a month
func (m *MockBudgetAllocationRepository) GetSpendingByCategory(workspaceID int32, year, month int) ([]*domain.CategorySpending, error) {
	if m.GetSpendingByCategoryFn != nil {
		return m.GetSpendingByCategoryFn(workspaceID, year, month)
	}
	key := allocationMonthKey(workspaceID, year, month)
	spending := m.SpendingByCategory[key]
	if spending == nil {
		return []*domain.CategorySpending{}, nil
	}
	return spending, nil
}

// SetSpendingByCategory sets the spending by category for a month (helper for tests)
func (m *MockBudgetAllocationRepository) SetSpendingByCategory(workspaceID int32, year, month int, spending []*domain.CategorySpending) {
	key := allocationMonthKey(workspaceID, year, month)
	m.SpendingByCategory[key] = spending
}

// GetCategoryTransactions retrieves transactions for a specific category and month
func (m *MockBudgetAllocationRepository) GetCategoryTransactions(workspaceID int32, categoryID int32, year, month int) ([]*domain.CategoryTransaction, error) {
	if m.GetCategoryTransactionsFn != nil {
		return m.GetCategoryTransactionsFn(workspaceID, categoryID, year, month)
	}
	return []*domain.CategoryTransaction{}, nil
}

// AddAllocation adds an allocation to the mock repository (helper for tests)
func (m *MockBudgetAllocationRepository) AddAllocation(allocation *domain.BudgetAllocation) {
	key := allocationKey(allocation.WorkspaceID, allocation.CategoryID, allocation.Year, allocation.Month)
	monthKey := allocationMonthKey(allocation.WorkspaceID, allocation.Year, allocation.Month)
	m.Allocations[key] = allocation
	m.ByWorkspaceMonth[monthKey] = append(m.ByWorkspaceMonth[monthKey], allocation)
}

// CountAllocationsForMonth returns the count of allocations for a specific month
func (m *MockBudgetAllocationRepository) CountAllocationsForMonth(workspaceID int32, year, month int) (int64, error) {
	if m.CountAllocationsForMonthFn != nil {
		return m.CountAllocationsForMonthFn(workspaceID, year, month)
	}
	key := allocationMonthKey(workspaceID, year, month)
	// Check if a count was explicitly set for this month
	if count, ok := m.AllocationCounts[key]; ok {
		return count, nil
	}
	// Fall back to counting allocations in the ByWorkspaceMonth map
	allocations := m.ByWorkspaceMonth[key]
	return int64(len(allocations)), nil
}

// SetAllocationCount sets the allocation count for a month (helper for tests)
func (m *MockBudgetAllocationRepository) SetAllocationCount(workspaceID int32, year, month int, count int64) {
	key := allocationMonthKey(workspaceID, year, month)
	m.AllocationCounts[key] = count
}

// CopyAllocationsToMonth copies all allocations from one month to another
func (m *MockBudgetAllocationRepository) CopyAllocationsToMonth(workspaceID int32, fromYear, fromMonth, toYear, toMonth int) error {
	if m.CopyAllocationsToMonthFn != nil {
		return m.CopyAllocationsToMonthFn(workspaceID, fromYear, fromMonth, toYear, toMonth)
	}
	fromKey := allocationMonthKey(workspaceID, fromYear, fromMonth)
	fromAllocations := m.ByWorkspaceMonth[fromKey]

	for _, alloc := range fromAllocations {
		newAlloc := &domain.BudgetAllocation{
			WorkspaceID: alloc.WorkspaceID,
			CategoryID:  alloc.CategoryID,
			Year:        toYear,
			Month:       toMonth,
			Amount:      alloc.Amount,
		}
		_, err := m.Upsert(newAlloc)
		if err != nil {
			return err
		}
	}
	return nil
}

// MockRecurringRepository is a mock implementation of domain.RecurringRepository
type MockRecurringRepository struct {
	Recurring            map[int32]*domain.RecurringTransaction
	ByWorkspace          map[int32][]*domain.RecurringTransaction
	NextID               int32
	ExistingTransactions map[string]bool // Tracks recurring+year+month combos for idempotency testing
	CreateFn             func(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error)
	GetByIDFn            func(workspaceID int32, id int32) (*domain.RecurringTransaction, error)
	ListFn               func(workspaceID int32, activeOnly *bool) ([]*domain.RecurringTransaction, error)
	UpdateFn             func(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error)
	DeleteFn             func(workspaceID int32, id int32) error
}

// NewMockRecurringRepository creates a new MockRecurringRepository
func NewMockRecurringRepository() *MockRecurringRepository {
	return &MockRecurringRepository{
		Recurring:   make(map[int32]*domain.RecurringTransaction),
		ByWorkspace: make(map[int32][]*domain.RecurringTransaction),
		NextID:      1,
	}
}

// Create creates a new recurring transaction
func (m *MockRecurringRepository) Create(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error) {
	if m.CreateFn != nil {
		return m.CreateFn(rt)
	}
	rt.ID = m.NextID
	m.NextID++
	rt.CreatedAt = time.Now()
	rt.UpdatedAt = time.Now()
	m.Recurring[rt.ID] = rt
	m.ByWorkspace[rt.WorkspaceID] = append(m.ByWorkspace[rt.WorkspaceID], rt)
	return rt, nil
}

// GetByID retrieves a recurring transaction by ID
func (m *MockRecurringRepository) GetByID(workspaceID int32, id int32) (*domain.RecurringTransaction, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	rt, ok := m.Recurring[id]
	if !ok || rt.WorkspaceID != workspaceID {
		return nil, domain.ErrRecurringNotFound
	}
	if rt.DeletedAt != nil {
		return nil, domain.ErrRecurringNotFound
	}
	return rt, nil
}

// ListByWorkspace retrieves all recurring transactions for a workspace
func (m *MockRecurringRepository) ListByWorkspace(workspaceID int32, activeOnly *bool) ([]*domain.RecurringTransaction, error) {
	if m.ListFn != nil {
		return m.ListFn(workspaceID, activeOnly)
	}
	rts := m.ByWorkspace[workspaceID]
	if rts == nil {
		return []*domain.RecurringTransaction{}, nil
	}
	var result []*domain.RecurringTransaction
	for _, rt := range rts {
		if rt.DeletedAt != nil {
			continue
		}
		if activeOnly != nil && rt.IsActive != *activeOnly {
			continue
		}
		result = append(result, rt)
	}
	if result == nil {
		return []*domain.RecurringTransaction{}, nil
	}
	return result, nil
}

// Update updates a recurring transaction
func (m *MockRecurringRepository) Update(rt *domain.RecurringTransaction) (*domain.RecurringTransaction, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(rt)
	}
	existing, ok := m.Recurring[rt.ID]
	if !ok || existing.WorkspaceID != rt.WorkspaceID {
		return nil, domain.ErrRecurringNotFound
	}
	if existing.DeletedAt != nil {
		return nil, domain.ErrRecurringNotFound
	}
	rt.UpdatedAt = time.Now()
	m.Recurring[rt.ID] = rt
	// Update in workspace list
	for i, r := range m.ByWorkspace[rt.WorkspaceID] {
		if r.ID == rt.ID {
			m.ByWorkspace[rt.WorkspaceID][i] = rt
			break
		}
	}
	return rt, nil
}

// Delete soft-deletes a recurring transaction
func (m *MockRecurringRepository) Delete(workspaceID int32, id int32) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(workspaceID, id)
	}
	rt, ok := m.Recurring[id]
	if !ok || rt.WorkspaceID != workspaceID {
		return domain.ErrRecurringNotFound
	}
	if rt.DeletedAt != nil {
		return domain.ErrRecurringNotFound
	}
	now := time.Now()
	rt.DeletedAt = &now
	return nil
}

// AddRecurring adds a recurring transaction to the mock repository (helper for tests)
func (m *MockRecurringRepository) AddRecurring(rt *domain.RecurringTransaction) {
	m.Recurring[rt.ID] = rt
	m.ByWorkspace[rt.WorkspaceID] = append(m.ByWorkspace[rt.WorkspaceID], rt)
}

// CheckTransactionExistsFn allows tests to override the behavior
var CheckTransactionExistsFn func(recurringID, workspaceID int32, year, month int) (bool, error)

// ExistingRecurringTransactions tracks which recurring+month combos exist (for idempotency testing)
// Key format: "recurringID-year-month"
func (m *MockRecurringRepository) SetTransactionExists(recurringID int32, year, month int) {
	if m.ExistingTransactions == nil {
		m.ExistingTransactions = make(map[string]bool)
	}
	key := fmt.Sprintf("%d-%d-%d", recurringID, year, month)
	m.ExistingTransactions[key] = true
}

// CheckTransactionExists checks if a transaction already exists for a recurring template in a specific month
func (m *MockRecurringRepository) CheckTransactionExists(recurringID, workspaceID int32, year, month int) (bool, error) {
	if CheckTransactionExistsFn != nil {
		return CheckTransactionExistsFn(recurringID, workspaceID, year, month)
	}
	if m.ExistingTransactions == nil {
		return false, nil
	}
	key := fmt.Sprintf("%d-%d-%d", recurringID, year, month)
	return m.ExistingTransactions[key], nil
}

// MockLoanProviderRepository is a mock implementation of domain.LoanProviderRepository
type MockLoanProviderRepository struct {
	Providers   map[int32]*domain.LoanProvider
	ByWorkspace map[int32][]*domain.LoanProvider
	NextID      int32
	CreateFn    func(provider *domain.LoanProvider) (*domain.LoanProvider, error)
	GetByIDFn   func(workspaceID int32, id int32) (*domain.LoanProvider, error)
	GetAllFn    func(workspaceID int32) ([]*domain.LoanProvider, error)
	UpdateFn    func(provider *domain.LoanProvider) (*domain.LoanProvider, error)
	DeleteFn    func(workspaceID int32, id int32) error
}

// NewMockLoanProviderRepository creates a new MockLoanProviderRepository
func NewMockLoanProviderRepository() *MockLoanProviderRepository {
	return &MockLoanProviderRepository{
		Providers:   make(map[int32]*domain.LoanProvider),
		ByWorkspace: make(map[int32][]*domain.LoanProvider),
		NextID:      1,
	}
}

// Create creates a new loan provider
func (m *MockLoanProviderRepository) Create(provider *domain.LoanProvider) (*domain.LoanProvider, error) {
	if m.CreateFn != nil {
		return m.CreateFn(provider)
	}
	provider.ID = m.NextID
	m.NextID++
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()
	m.Providers[provider.ID] = provider
	m.ByWorkspace[provider.WorkspaceID] = append(m.ByWorkspace[provider.WorkspaceID], provider)
	return provider, nil
}

// GetByID retrieves a loan provider by ID
func (m *MockLoanProviderRepository) GetByID(workspaceID int32, id int32) (*domain.LoanProvider, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	provider, ok := m.Providers[id]
	if !ok || provider.WorkspaceID != workspaceID {
		return nil, domain.ErrLoanProviderNotFound
	}
	if provider.DeletedAt != nil {
		return nil, domain.ErrLoanProviderNotFound
	}
	return provider, nil
}

// GetAllByWorkspace retrieves all loan providers for a workspace
func (m *MockLoanProviderRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.LoanProvider, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(workspaceID)
	}
	providers := m.ByWorkspace[workspaceID]
	if providers == nil {
		return []*domain.LoanProvider{}, nil
	}
	var result []*domain.LoanProvider
	for _, p := range providers {
		if p.DeletedAt == nil {
			result = append(result, p)
		}
	}
	if result == nil {
		return []*domain.LoanProvider{}, nil
	}
	return result, nil
}

// Update updates a loan provider
func (m *MockLoanProviderRepository) Update(provider *domain.LoanProvider) (*domain.LoanProvider, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(provider)
	}
	existing, ok := m.Providers[provider.ID]
	if !ok || existing.WorkspaceID != provider.WorkspaceID {
		return nil, domain.ErrLoanProviderNotFound
	}
	if existing.DeletedAt != nil {
		return nil, domain.ErrLoanProviderNotFound
	}
	provider.UpdatedAt = time.Now()
	m.Providers[provider.ID] = provider
	// Update in workspace list
	for i, p := range m.ByWorkspace[provider.WorkspaceID] {
		if p.ID == provider.ID {
			m.ByWorkspace[provider.WorkspaceID][i] = provider
			break
		}
	}
	return provider, nil
}

// SoftDelete soft-deletes a loan provider
func (m *MockLoanProviderRepository) SoftDelete(workspaceID int32, id int32) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(workspaceID, id)
	}
	provider, ok := m.Providers[id]
	if !ok || provider.WorkspaceID != workspaceID {
		return domain.ErrLoanProviderNotFound
	}
	if provider.DeletedAt != nil {
		return domain.ErrLoanProviderNotFound
	}
	now := time.Now()
	provider.DeletedAt = &now
	return nil
}

// AddLoanProvider adds a loan provider to the mock repository (helper for tests)
func (m *MockLoanProviderRepository) AddLoanProvider(provider *domain.LoanProvider) {
	m.Providers[provider.ID] = provider
	m.ByWorkspace[provider.WorkspaceID] = append(m.ByWorkspace[provider.WorkspaceID], provider)
}

// MockLoanRepository is a mock implementation of domain.LoanRepository
type MockLoanRepository struct {
	Loans              map[int32]*domain.Loan
	ByWorkspace        map[int32][]*domain.Loan
	ActiveLoans        map[string][]*domain.Loan
	CompletedLoans     map[string][]*domain.Loan
	ActiveLoanCounts   map[string]int64
	LoansWithStats     []*domain.LoanWithStats
	ActiveWithStats    []*domain.LoanWithStats
	CompletedWithStats []*domain.LoanWithStats
	NextID             int32
	CreateFn           func(loan *domain.Loan) (*domain.Loan, error)
	GetByIDFn          func(workspaceID int32, id int32) (*domain.Loan, error)
	GetAllFn           func(workspaceID int32) ([]*domain.Loan, error)
	GetActiveFn        func(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error)
	GetCompletedFn     func(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error)
	UpdateFn           func(loan *domain.Loan) (*domain.Loan, error)
	DeleteFn           func(workspaceID int32, id int32) error
	CountActiveFn      func(workspaceID int32, providerID int32, currentYear, currentMonth int) (int64, error)
}

// NewMockLoanRepository creates a new MockLoanRepository
func NewMockLoanRepository() *MockLoanRepository {
	return &MockLoanRepository{
		Loans:            make(map[int32]*domain.Loan),
		ByWorkspace:      make(map[int32][]*domain.Loan),
		ActiveLoans:      make(map[string][]*domain.Loan),
		CompletedLoans:   make(map[string][]*domain.Loan),
		ActiveLoanCounts: make(map[string]int64),
		NextID:           1,
	}
}

func loanMonthKey(workspaceID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d", workspaceID, year, month)
}

func loanProviderMonthKey(workspaceID, providerID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d-%d", workspaceID, providerID, year, month)
}

// Create creates a new loan
func (m *MockLoanRepository) Create(loan *domain.Loan) (*domain.Loan, error) {
	if m.CreateFn != nil {
		return m.CreateFn(loan)
	}
	loan.ID = m.NextID
	m.NextID++
	loan.CreatedAt = time.Now()
	loan.UpdatedAt = time.Now()
	m.Loans[loan.ID] = loan
	m.ByWorkspace[loan.WorkspaceID] = append(m.ByWorkspace[loan.WorkspaceID], loan)
	return loan, nil
}

// CreateTx creates a new loan within a transaction (mock just calls Create)
func (m *MockLoanRepository) CreateTx(tx interface{}, loan *domain.Loan) (*domain.Loan, error) {
	return m.Create(loan)
}

// GetByID retrieves a loan by ID
func (m *MockLoanRepository) GetByID(workspaceID int32, id int32) (*domain.Loan, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(workspaceID, id)
	}
	loan, ok := m.Loans[id]
	if !ok || loan.WorkspaceID != workspaceID {
		return nil, domain.ErrLoanNotFound
	}
	if loan.DeletedAt != nil {
		return nil, domain.ErrLoanNotFound
	}
	return loan, nil
}

// GetAllByWorkspace retrieves all loans for a workspace
func (m *MockLoanRepository) GetAllByWorkspace(workspaceID int32) ([]*domain.Loan, error) {
	if m.GetAllFn != nil {
		return m.GetAllFn(workspaceID)
	}
	loans := m.ByWorkspace[workspaceID]
	if loans == nil {
		return []*domain.Loan{}, nil
	}
	var result []*domain.Loan
	for _, l := range loans {
		if l.DeletedAt == nil {
			result = append(result, l)
		}
	}
	if result == nil {
		return []*domain.Loan{}, nil
	}
	return result, nil
}

// GetActiveByWorkspace retrieves active loans for a workspace
func (m *MockLoanRepository) GetActiveByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	if m.GetActiveFn != nil {
		return m.GetActiveFn(workspaceID, currentYear, currentMonth)
	}
	key := loanMonthKey(workspaceID, currentYear, currentMonth)
	if loans, ok := m.ActiveLoans[key]; ok {
		return loans, nil
	}
	// Calculate active loans from all loans
	allLoans := m.ByWorkspace[workspaceID]
	var result []*domain.Loan
	for _, l := range allLoans {
		if l.DeletedAt == nil && l.IsActive(currentYear, currentMonth) {
			result = append(result, l)
		}
	}
	if result == nil {
		return []*domain.Loan{}, nil
	}
	return result, nil
}

// GetCompletedByWorkspace retrieves completed loans for a workspace
func (m *MockLoanRepository) GetCompletedByWorkspace(workspaceID int32, currentYear, currentMonth int) ([]*domain.Loan, error) {
	if m.GetCompletedFn != nil {
		return m.GetCompletedFn(workspaceID, currentYear, currentMonth)
	}
	key := loanMonthKey(workspaceID, currentYear, currentMonth)
	if loans, ok := m.CompletedLoans[key]; ok {
		return loans, nil
	}
	// Calculate completed loans from all loans
	allLoans := m.ByWorkspace[workspaceID]
	var result []*domain.Loan
	for _, l := range allLoans {
		if l.DeletedAt == nil && !l.IsActive(currentYear, currentMonth) {
			result = append(result, l)
		}
	}
	if result == nil {
		return []*domain.Loan{}, nil
	}
	return result, nil
}

// Update updates a loan
func (m *MockLoanRepository) Update(loan *domain.Loan) (*domain.Loan, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(loan)
	}
	existing, ok := m.Loans[loan.ID]
	if !ok || existing.WorkspaceID != loan.WorkspaceID {
		return nil, domain.ErrLoanNotFound
	}
	if existing.DeletedAt != nil {
		return nil, domain.ErrLoanNotFound
	}
	loan.UpdatedAt = time.Now()
	m.Loans[loan.ID] = loan
	// Update in workspace list
	for i, l := range m.ByWorkspace[loan.WorkspaceID] {
		if l.ID == loan.ID {
			m.ByWorkspace[loan.WorkspaceID][i] = loan
			break
		}
	}
	return loan, nil
}

// UpdatePartial updates only itemName and notes of a loan
func (m *MockLoanRepository) UpdatePartial(workspaceID int32, id int32, itemName string, notes *string) (*domain.Loan, error) {
	loan, ok := m.Loans[id]
	if !ok || loan.WorkspaceID != workspaceID {
		return nil, domain.ErrLoanNotFound
	}
	if loan.DeletedAt != nil {
		return nil, domain.ErrLoanNotFound
	}
	loan.ItemName = itemName
	loan.Notes = notes
	loan.UpdatedAt = time.Now()
	return loan, nil
}

// SoftDelete soft-deletes a loan
func (m *MockLoanRepository) SoftDelete(workspaceID int32, id int32) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(workspaceID, id)
	}
	loan, ok := m.Loans[id]
	if !ok || loan.WorkspaceID != workspaceID {
		return domain.ErrLoanNotFound
	}
	if loan.DeletedAt != nil {
		return domain.ErrLoanNotFound
	}
	now := time.Now()
	loan.DeletedAt = &now
	return nil
}

// CountActiveLoansByProvider counts active loans for a provider
func (m *MockLoanRepository) CountActiveLoansByProvider(workspaceID int32, providerID int32, currentYear, currentMonth int) (int64, error) {
	if m.CountActiveFn != nil {
		return m.CountActiveFn(workspaceID, providerID, currentYear, currentMonth)
	}
	key := loanProviderMonthKey(workspaceID, providerID, currentYear, currentMonth)
	if count, ok := m.ActiveLoanCounts[key]; ok {
		return count, nil
	}
	// Calculate from all loans
	allLoans := m.ByWorkspace[workspaceID]
	var count int64
	for _, l := range allLoans {
		if l.DeletedAt == nil && l.ProviderID == providerID && l.IsActive(currentYear, currentMonth) {
			count++
		}
	}
	return count, nil
}

// AddLoan adds a loan to the mock repository (helper for tests)
func (m *MockLoanRepository) AddLoan(loan *domain.Loan) {
	m.Loans[loan.ID] = loan
	m.ByWorkspace[loan.WorkspaceID] = append(m.ByWorkspace[loan.WorkspaceID], loan)
}

// SetActiveLoans sets the active loans for testing (helper for tests)
func (m *MockLoanRepository) SetActiveLoans(workspaceID int32, year, month int, loans []*domain.Loan) {
	key := loanMonthKey(workspaceID, year, month)
	m.ActiveLoans[key] = loans
}

// SetCompletedLoans sets the completed loans for testing (helper for tests)
func (m *MockLoanRepository) SetCompletedLoans(workspaceID int32, year, month int, loans []*domain.Loan) {
	key := loanMonthKey(workspaceID, year, month)
	m.CompletedLoans[key] = loans
}

// SetActiveLoanCount sets the active loan count for a provider (helper for tests)
func (m *MockLoanRepository) SetActiveLoanCount(workspaceID, providerID int32, year, month int, count int64) {
	key := loanProviderMonthKey(workspaceID, providerID, year, month)
	m.ActiveLoanCounts[key] = count
}

// GetAllWithStats retrieves all loans with payment statistics
func (m *MockLoanRepository) GetAllWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	if m.LoansWithStats != nil {
		return m.LoansWithStats, nil
	}
	return []*domain.LoanWithStats{}, nil
}

// GetActiveWithStats retrieves active loans with payment statistics
func (m *MockLoanRepository) GetActiveWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	if m.ActiveWithStats != nil {
		return m.ActiveWithStats, nil
	}
	return []*domain.LoanWithStats{}, nil
}

// GetCompletedWithStats retrieves completed loans with payment statistics
func (m *MockLoanRepository) GetCompletedWithStats(workspaceID int32) ([]*domain.LoanWithStats, error) {
	if m.CompletedWithStats != nil {
		return m.CompletedWithStats, nil
	}
	return []*domain.LoanWithStats{}, nil
}

// SetLoansWithStats sets the loans with stats for testing (helper for tests)
func (m *MockLoanRepository) SetLoansWithStats(loans []*domain.LoanWithStats) {
	m.LoansWithStats = loans
}

// SetActiveWithStats sets the active loans with stats for testing (helper for tests)
func (m *MockLoanRepository) SetActiveWithStats(loans []*domain.LoanWithStats) {
	m.ActiveWithStats = loans
}

// SetCompletedWithStats sets the completed loans with stats for testing (helper for tests)
func (m *MockLoanRepository) SetCompletedWithStats(loans []*domain.LoanWithStats) {
	m.CompletedWithStats = loans
}

// MockLoanPaymentRepository is a mock implementation of domain.LoanPaymentRepository
type MockLoanPaymentRepository struct {
	Payments    map[int32]*domain.LoanPayment
	ByLoanID    map[int32][]*domain.LoanPayment
	ByMonth     map[string][]*domain.LoanPayment
	NextID      int32
	CreateFn    func(payment *domain.LoanPayment) (*domain.LoanPayment, error)
	GetByIDFn   func(id int32) (*domain.LoanPayment, error)
}

// NewMockLoanPaymentRepository creates a new MockLoanPaymentRepository
func NewMockLoanPaymentRepository() *MockLoanPaymentRepository {
	return &MockLoanPaymentRepository{
		Payments: make(map[int32]*domain.LoanPayment),
		ByLoanID: make(map[int32][]*domain.LoanPayment),
		ByMonth:  make(map[string][]*domain.LoanPayment),
		NextID:   1,
	}
}

func paymentMonthKey(workspaceID int32, year, month int) string {
	return fmt.Sprintf("%d-%d-%d", workspaceID, year, month)
}

// Create creates a new loan payment
func (m *MockLoanPaymentRepository) Create(payment *domain.LoanPayment) (*domain.LoanPayment, error) {
	if m.CreateFn != nil {
		return m.CreateFn(payment)
	}
	payment.ID = m.NextID
	m.NextID++
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()
	m.Payments[payment.ID] = payment
	m.ByLoanID[payment.LoanID] = append(m.ByLoanID[payment.LoanID], payment)
	return payment, nil
}

// CreateBatch creates multiple loan payments
func (m *MockLoanPaymentRepository) CreateBatch(payments []*domain.LoanPayment) error {
	for _, payment := range payments {
		_, err := m.Create(payment)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateBatchTx creates multiple loan payments within a transaction (mock just calls CreateBatch)
func (m *MockLoanPaymentRepository) CreateBatchTx(tx interface{}, payments []*domain.LoanPayment) error {
	return m.CreateBatch(payments)
}

// GetByID retrieves a loan payment by ID
func (m *MockLoanPaymentRepository) GetByID(id int32) (*domain.LoanPayment, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}
	payment, ok := m.Payments[id]
	if !ok {
		return nil, domain.ErrLoanPaymentNotFound
	}
	return payment, nil
}

// GetByLoanID retrieves all payments for a loan
func (m *MockLoanPaymentRepository) GetByLoanID(loanID int32) ([]*domain.LoanPayment, error) {
	payments := m.ByLoanID[loanID]
	if payments == nil {
		return []*domain.LoanPayment{}, nil
	}
	return payments, nil
}

// GetByLoanAndNumber retrieves a specific payment by loan ID and payment number
func (m *MockLoanPaymentRepository) GetByLoanAndNumber(loanID int32, paymentNumber int32) (*domain.LoanPayment, error) {
	payments := m.ByLoanID[loanID]
	for _, p := range payments {
		if p.PaymentNumber == paymentNumber {
			return p, nil
		}
	}
	return nil, domain.ErrLoanPaymentNotFound
}

// UpdateAmount updates the amount of a specific payment
func (m *MockLoanPaymentRepository) UpdateAmount(id int32, amount decimal.Decimal) (*domain.LoanPayment, error) {
	payment, ok := m.Payments[id]
	if !ok {
		return nil, domain.ErrLoanPaymentNotFound
	}
	payment.Amount = amount
	payment.UpdatedAt = time.Now()
	return payment, nil
}

// TogglePaid toggles the paid status of a payment
func (m *MockLoanPaymentRepository) TogglePaid(id int32, paid bool, paidDate *time.Time) (*domain.LoanPayment, error) {
	payment, ok := m.Payments[id]
	if !ok {
		return nil, domain.ErrLoanPaymentNotFound
	}
	payment.Paid = paid
	payment.PaidDate = paidDate
	payment.UpdatedAt = time.Now()
	return payment, nil
}

// GetByMonth retrieves all loan payments due in a specific month
func (m *MockLoanPaymentRepository) GetByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	key := paymentMonthKey(workspaceID, year, month)
	payments := m.ByMonth[key]
	if payments == nil {
		return []*domain.LoanPayment{}, nil
	}
	return payments, nil
}

// GetUnpaidByMonth retrieves unpaid loan payments due in a specific month
func (m *MockLoanPaymentRepository) GetUnpaidByMonth(workspaceID int32, year, month int) ([]*domain.LoanPayment, error) {
	key := paymentMonthKey(workspaceID, year, month)
	payments := m.ByMonth[key]
	var unpaid []*domain.LoanPayment
	for _, p := range payments {
		if !p.Paid {
			unpaid = append(unpaid, p)
		}
	}
	return unpaid, nil
}

// AddPayment adds a payment to the mock repository (helper for tests)
func (m *MockLoanPaymentRepository) AddPayment(payment *domain.LoanPayment) {
	m.Payments[payment.ID] = payment
	m.ByLoanID[payment.LoanID] = append(m.ByLoanID[payment.LoanID], payment)
}

// SetPaymentsByMonth sets payments for a specific month (helper for tests)
func (m *MockLoanPaymentRepository) SetPaymentsByMonth(workspaceID int32, year, month int, payments []*domain.LoanPayment) {
	key := paymentMonthKey(workspaceID, year, month)
	m.ByMonth[key] = payments
}

// GetDeleteStats retrieves payment statistics for a loan
func (m *MockLoanPaymentRepository) GetDeleteStats(loanID int32) (*domain.LoanDeleteStats, error) {
	payments := m.ByLoanID[loanID]
	stats := &domain.LoanDeleteStats{
		TotalCount:  0,
		PaidCount:   0,
		UnpaidCount: 0,
		TotalAmount: decimal.Zero,
	}
	for _, p := range payments {
		stats.TotalCount++
		if p.Paid {
			stats.PaidCount++
		} else {
			stats.UnpaidCount++
		}
		stats.TotalAmount = stats.TotalAmount.Add(p.Amount)
	}
	return stats, nil
}
