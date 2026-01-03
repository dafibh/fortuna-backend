package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// WishlistNoteService handles wishlist item note business logic
type WishlistNoteService struct {
	noteRepo domain.WishlistNoteRepository
	itemRepo domain.WishlistItemRepository
}

// NewWishlistNoteService creates a new WishlistNoteService
func NewWishlistNoteService(noteRepo domain.WishlistNoteRepository, itemRepo domain.WishlistItemRepository) *WishlistNoteService {
	return &WishlistNoteService{
		noteRepo: noteRepo,
		itemRepo: itemRepo,
	}
}

// CreateNoteInput contains input for creating a note
type CreateNoteInput struct {
	Content string
}

// CreateNote creates a new note for an item
func (s *WishlistNoteService) CreateNote(workspaceID int32, itemID int32, input CreateNoteInput) (*domain.WishlistItemNote, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	// Validate content
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, domain.ErrNoteContentEmpty
	}

	note := &domain.WishlistItemNote{
		ItemID:  itemID,
		Content: content,
	}

	return s.noteRepo.Create(note)
}

// GetNoteByID retrieves a note by ID
func (s *WishlistNoteService) GetNoteByID(workspaceID int32, id int32) (*domain.WishlistItemNote, error) {
	return s.noteRepo.GetByID(workspaceID, id)
}

// ListNotesByItem retrieves all notes for an item
func (s *WishlistNoteService) ListNotesByItem(workspaceID int32, itemID int32, sortAsc bool) ([]*domain.WishlistItemNote, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	return s.noteRepo.ListByItem(workspaceID, itemID, sortAsc)
}

// CountNotesByItem counts notes for an item
func (s *WishlistNoteService) CountNotesByItem(workspaceID int32, itemID int32) (int64, error) {
	// Verify item exists and belongs to workspace
	_, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return 0, err
	}

	return s.noteRepo.CountByItem(workspaceID, itemID)
}

// UpdateNote updates a note's content
func (s *WishlistNoteService) UpdateNote(workspaceID int32, id int32, content string) (*domain.WishlistItemNote, error) {
	// Verify note exists
	_, err := s.noteRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Validate content
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrNoteContentEmpty
	}

	return s.noteRepo.Update(workspaceID, id, content)
}

// DeleteNote soft-deletes a note
func (s *WishlistNoteService) DeleteNote(workspaceID int32, id int32) error {
	// Verify note exists
	_, err := s.noteRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	return s.noteRepo.Delete(workspaceID, id)
}
