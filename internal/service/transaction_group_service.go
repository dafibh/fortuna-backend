package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocket event payloads for transaction group operations
type GroupCreatedPayload struct {
	ID           int32  `json:"id"`
	Name         string `json:"name"`
	Month        string `json:"month"`
	ChildCount   int32  `json:"childCount"`
	TotalAmount  string `json:"totalAmount"`
	AutoDetected bool   `json:"autoDetected"`
}

type GroupUpdatedPayload struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type GroupDeletedPayload struct {
	ID   int32  `json:"id"`
	Mode string `json:"mode"` // "ungroup", "delete_all", "auto_empty"
}

type GroupChildrenChangedPayload struct {
	ID          int32  `json:"id"`
	ChildCount  int32  `json:"childCount"`
	TotalAmount string `json:"totalAmount"`
}

// TransactionGroupService handles business logic for transaction grouping
type TransactionGroupService struct {
	transactionGroupRepo domain.TransactionGroupRepository
	transactionRepo      domain.TransactionRepository
	eventPublisher       websocket.EventPublisher
}

// NewTransactionGroupService creates a new TransactionGroupService
func NewTransactionGroupService(
	groupRepo domain.TransactionGroupRepository,
	transactionRepo domain.TransactionRepository,
) *TransactionGroupService {
	return &TransactionGroupService{
		transactionGroupRepo: groupRepo,
		transactionRepo:      transactionRepo,
	}
}

// SetEventPublisher sets the WebSocket event publisher
func (s *TransactionGroupService) SetEventPublisher(publisher websocket.EventPublisher) {
	s.eventPublisher = publisher
}

func (s *TransactionGroupService) publishEvent(workspaceID int32, event websocket.Event) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(workspaceID, event)
	}
}

// CreateGroup creates a new transaction group with the given transactions
func (s *TransactionGroupService) CreateGroup(workspaceID int32, name string, transactionIDs []int32) (*domain.TransactionGroup, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrGroupNameEmpty
	}

	// Fetch all transactions and verify workspace ownership
	transactions, err := s.transactionRepo.GetByIDs(workspaceID, transactionIDs)
	if err != nil {
		return nil, err
	}
	if len(transactions) != len(transactionIDs) {
		return nil, domain.ErrTransactionNotFound
	}

	// Validate all transactions are in the same month
	var month string
	for i, tx := range transactions {
		txMonth := tx.TransactionDate.Format("2006-01")
		if i == 0 {
			month = txMonth
		} else if txMonth != month {
			return nil, domain.ErrMonthBoundaryViolation
		}
	}

	// Validate no transaction is already in a group
	for _, tx := range transactions {
		if tx.GroupID != nil {
			return nil, domain.ErrAlreadyGrouped
		}
	}

	// Create group and assign transactions atomically
	group := &domain.TransactionGroup{
		WorkspaceID:  workspaceID,
		Name:         name,
		Month:        month,
		AutoDetected: false,
	}

	created, err := s.transactionGroupRepo.CreateWithAssignment(group, transactionIDs)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", created.ID).
		Str("name", created.Name).
		Int("transaction_count", len(transactionIDs)).
		Msg("Transaction group created")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupCreated(GroupCreatedPayload{
		ID:           created.ID,
		Name:         created.Name,
		Month:        created.Month,
		ChildCount:   created.ChildCount,
		TotalAmount:  created.TotalAmount.StringFixed(2),
		AutoDetected: created.AutoDetected,
	}))

	return created, nil
}

// AddTransactionsToGroup adds transactions to an existing group
func (s *TransactionGroupService) AddTransactionsToGroup(workspaceID int32, groupID int32, transactionIDs []int32) (*domain.TransactionGroup, error) {
	// Validate group exists and belongs to workspace
	group, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	// Fetch all transactions and verify workspace ownership
	transactions, err := s.transactionRepo.GetByIDs(workspaceID, transactionIDs)
	if err != nil {
		return nil, err
	}
	if len(transactions) != len(transactionIDs) {
		return nil, domain.ErrTransactionNotFound
	}

	// Validate all transactions are in the same month as the group
	for _, tx := range transactions {
		txMonth := tx.TransactionDate.Format("2006-01")
		if txMonth != group.Month {
			return nil, domain.ErrMonthBoundaryViolation
		}
	}

	// Validate no transaction is already in a group
	for _, tx := range transactions {
		if tx.GroupID != nil {
			return nil, domain.ErrAlreadyGrouped
		}
	}

	// Assign transactions to group
	err = s.transactionGroupRepo.AssignGroupToTransactions(workspaceID, groupID, transactionIDs)
	if err != nil {
		return nil, err
	}

	// Fetch updated group with recalculated totals
	updated, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", groupID).
		Int("added_count", len(transactionIDs)).
		Msg("Transactions added to group")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupChildrenChanged(GroupChildrenChangedPayload{
		ID:          updated.ID,
		ChildCount:  updated.ChildCount,
		TotalAmount: updated.TotalAmount.StringFixed(2),
	}))

	return updated, nil
}

// RemoveTransactionsFromGroup removes transactions from a group and auto-deletes if empty
func (s *TransactionGroupService) RemoveTransactionsFromGroup(workspaceID int32, groupID int32, transactionIDs []int32) (*domain.TransactionGroup, bool, error) {
	// Validate group exists and belongs to workspace
	_, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, false, err
	}

	// Fetch all transactions and verify workspace ownership
	transactions, err := s.transactionRepo.GetByIDs(workspaceID, transactionIDs)
	if err != nil {
		return nil, false, err
	}
	if len(transactions) != len(transactionIDs) {
		return nil, false, domain.ErrTransactionNotFound
	}

	// Validate all transactions belong to this group
	for _, tx := range transactions {
		if tx.GroupID == nil || *tx.GroupID != groupID {
			return nil, false, domain.ErrTransactionNotInGroup
		}
	}

	// Unassign transactions from group
	err = s.transactionGroupRepo.UnassignGroupFromTransactions(workspaceID, transactionIDs)
	if err != nil {
		return nil, false, err
	}

	// Check remaining children
	updated, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, false, err
	}

	if updated.ChildCount == 0 {
		// Auto-delete empty group
		err = s.transactionGroupRepo.Delete(workspaceID, groupID)
		if err != nil {
			return nil, false, err
		}

		log.Info().
			Int32("workspace_id", workspaceID).
			Int32("group_id", groupID).
			Msg("Empty group auto-deleted after removing last children")

		s.publishEvent(workspaceID, websocket.TransactionGroupDeleted(GroupDeletedPayload{
			ID:   groupID,
			Mode: "auto_empty",
		}))

		return nil, true, nil
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", groupID).
		Int("removed_count", len(transactionIDs)).
		Msg("Transactions removed from group")

	// Publish children changed event
	s.publishEvent(workspaceID, websocket.TransactionGroupChildrenChanged(GroupChildrenChangedPayload{
		ID:          updated.ID,
		ChildCount:  updated.ChildCount,
		TotalAmount: updated.TotalAmount.StringFixed(2),
	}))

	return updated, false, nil
}

// UngroupGroup unassigns all children from a group and deletes the group record
func (s *TransactionGroupService) UngroupGroup(workspaceID int32, groupID int32) (*domain.GroupOperationResult, error) {
	// Validate group exists and belongs to workspace
	_, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	// Explicitly unassign all children (preferred over relying on ON DELETE SET NULL)
	count, err := s.transactionGroupRepo.UnassignAllFromGroup(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	// Delete the group record
	err = s.transactionGroupRepo.Delete(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", groupID).
		Int64("children_unassigned", count).
		Msg("Transaction group ungrouped")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupDeleted(GroupDeletedPayload{
		ID:   groupID,
		Mode: "ungroup",
	}))

	return &domain.GroupOperationResult{
		GroupID:          groupID,
		Mode:             "ungroup",
		ChildrenAffected: int32(count),
	}, nil
}

// DeleteGroupWithChildren atomically soft-deletes all children and hard-deletes the group
func (s *TransactionGroupService) DeleteGroupWithChildren(workspaceID int32, groupID int32) (*domain.GroupOperationResult, error) {
	// Validate group exists and belongs to workspace
	_, err := s.transactionGroupRepo.GetByID(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	// Atomic delete: soft-delete children + hard-delete group in single transaction
	count, err := s.transactionGroupRepo.DeleteGroupAndChildren(workspaceID, groupID)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", groupID).
		Int32("children_deleted", count).
		Msg("Transaction group and children deleted")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupDeleted(GroupDeletedPayload{
		ID:   groupID,
		Mode: "delete_all",
	}))

	return &domain.GroupOperationResult{
		GroupID:          groupID,
		Mode:             "delete_all",
		ChildrenAffected: count,
	}, nil
}

// GetGroupsByMonth returns all groups for a workspace and month
func (s *TransactionGroupService) GetGroupsByMonth(workspaceID int32, month string) ([]*domain.TransactionGroup, error) {
	return s.transactionGroupRepo.GetGroupsByMonth(workspaceID, month)
}

// EnsureAutoGroups detects consolidated_monthly providers with >=2 ungrouped transactions
// in the given month and auto-creates groups for them. This is fire-and-forget:
// errors are logged but never propagated to the caller.
func (s *TransactionGroupService) EnsureAutoGroups(workspaceID int32, month string) error {
	candidates, err := s.transactionGroupRepo.GetConsolidatedProvidersByMonth(workspaceID, month)
	if err != nil {
		log.Warn().Err(err).Int32("workspace_id", workspaceID).Str("month", month).Msg("auto-group: failed to get candidates")
		return nil
	}
	if len(candidates) == 0 {
		return nil
	}

	// Parse month to generate human-readable group name
	monthTime, err := time.Parse("2006-01", month)
	if err != nil {
		log.Warn().Err(err).Str("month", month).Msg("auto-group: failed to parse month")
		return nil
	}
	monthLabel := monthTime.Format("January 2006")

	for _, candidate := range candidates {
		s.ensureAutoGroupForProvider(workspaceID, month, monthLabel, candidate)
	}

	return nil
}

func (s *TransactionGroupService) ensureAutoGroupForProvider(workspaceID int32, month string, monthLabel string, candidate domain.AutoDetectionCandidate) {
	// Check for existing auto-detected group (idempotency)
	existingGroup, err := s.transactionGroupRepo.GetAutoDetectedGroupByProviderMonth(workspaceID, candidate.ProviderID, month)
	if err != nil && err != domain.ErrGroupNotFound {
		log.Warn().Err(err).Int32("provider_id", candidate.ProviderID).Msg("auto-group: failed to check existing group")
		return
	}

	// Get ungrouped transaction IDs
	txIDs, err := s.transactionGroupRepo.GetUngroupedTransactionIDsByProviderMonth(workspaceID, candidate.ProviderID, month)
	if err != nil {
		log.Warn().Err(err).Int32("provider_id", candidate.ProviderID).Msg("auto-group: failed to get ungrouped tx IDs")
		return
	}
	if len(txIDs) == 0 {
		return
	}

	if existingGroup != nil {
		// Add new ungrouped transactions to the existing group
		err = s.transactionGroupRepo.AssignGroupToTransactions(workspaceID, existingGroup.ID, txIDs)
		if err != nil {
			log.Warn().Err(err).Int32("group_id", existingGroup.ID).Msg("auto-group: failed to assign to existing group")
			return
		}
		log.Info().
			Int32("workspace_id", workspaceID).
			Int32("group_id", existingGroup.ID).
			Int("added_count", len(txIDs)).
			Msg("auto-group: added transactions to existing group")
		return
	}

	// Create new auto-detected group
	groupName := fmt.Sprintf("%s - %s", candidate.ProviderName, monthLabel)
	providerID := candidate.ProviderID
	group := &domain.TransactionGroup{
		WorkspaceID:    workspaceID,
		Name:           groupName,
		Month:          month,
		AutoDetected:   true,
		LoanProviderID: &providerID,
	}

	created, err := s.transactionGroupRepo.Create(group)
	if err != nil {
		log.Warn().Err(err).Str("name", groupName).Msg("auto-group: failed to create group")
		return
	}

	// Assign transactions to the new group
	err = s.transactionGroupRepo.AssignGroupToTransactions(workspaceID, created.ID, txIDs)
	if err != nil {
		log.Warn().Err(err).Int32("group_id", created.ID).Msg("auto-group: failed to assign transactions")
		return
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", created.ID).
		Str("name", groupName).
		Int("transaction_count", len(txIDs)).
		Msg("auto-group: created new group")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupCreated(GroupCreatedPayload{
		ID:           created.ID,
		Name:         created.Name,
		Month:        created.Month,
		ChildCount:   int32(len(txIDs)),
		AutoDetected: true,
	}))
}

// RenameGroup renames a transaction group
func (s *TransactionGroupService) RenameGroup(workspaceID int32, groupID int32, name string) (*domain.TransactionGroup, error) {
	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrGroupNameEmpty
	}

	// Validate group exists and belongs to workspace (implicit in UpdateName)
	updated, err := s.transactionGroupRepo.UpdateName(workspaceID, groupID, name)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int32("workspace_id", workspaceID).
		Int32("group_id", groupID).
		Str("name", name).
		Msg("Transaction group renamed")

	// Publish WebSocket event
	s.publishEvent(workspaceID, websocket.TransactionGroupUpdated(GroupUpdatedPayload{
		ID:   updated.ID,
		Name: updated.Name,
	}))

	return updated, nil
}
