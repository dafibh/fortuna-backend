package testutil

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/google/uuid"
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
	Accounts      map[int32]*domain.Account
	ByWorkspace   map[int32][]*domain.Account
	NextID        int32
	CreateFn      func(account *domain.Account) (*domain.Account, error)
	GetByIDFn     func(workspaceID int32, id int32) (*domain.Account, error)
	GetAllFn      func(workspaceID int32, includeArchived bool) ([]*domain.Account, error)
	UpdateFn      func(workspaceID int32, id int32, name string) (*domain.Account, error)
	SoftDeleteFn  func(workspaceID int32, id int32) error
	HardDeleteFn  func(workspaceID int32, id int32) error
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

// MockTransactionRepository is a mock implementation of domain.TransactionRepository
type MockTransactionRepository struct {
	Transactions  map[int32]*domain.Transaction
	ByWorkspace   map[int32][]*domain.Transaction
	NextID        int32
	CreateFn      func(transaction *domain.Transaction) (*domain.Transaction, error)
	GetByIDFn     func(workspaceID int32, id int32) (*domain.Transaction, error)
	GetByWSFn     func(workspaceID int32, filters *domain.TransactionFilters) (*domain.PaginatedTransactions, error)
}

// NewMockTransactionRepository creates a new MockTransactionRepository
func NewMockTransactionRepository() *MockTransactionRepository {
	return &MockTransactionRepository{
		Transactions: make(map[int32]*domain.Transaction),
		ByWorkspace:  make(map[int32][]*domain.Transaction),
		NextID:       1,
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

// AddTransaction adds a transaction to the mock repository (helper for tests)
func (m *MockTransactionRepository) AddTransaction(transaction *domain.Transaction) {
	m.Transactions[transaction.ID] = transaction
	m.ByWorkspace[transaction.WorkspaceID] = append(m.ByWorkspace[transaction.WorkspaceID], transaction)
}
