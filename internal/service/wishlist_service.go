package service

import (
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// WishlistService handles wishlist business logic
type WishlistService struct {
	wishlistRepo domain.WishlistRepository
}

// NewWishlistService creates a new WishlistService
func NewWishlistService(wishlistRepo domain.WishlistRepository) *WishlistService {
	return &WishlistService{wishlistRepo: wishlistRepo}
}

// CreateWishlistInput contains input for creating a wishlist
type CreateWishlistInput struct {
	Name string
}

// CreateWishlist creates a new wishlist
func (s *WishlistService) CreateWishlist(workspaceID int32, input CreateWishlistInput) (*domain.Wishlist, error) {
	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrWishlistNameEmpty
	}
	if len(name) > 255 {
		return nil, domain.ErrWishlistNameTooLong
	}

	wishlist := &domain.Wishlist{
		WorkspaceID: workspaceID,
		Name:        name,
	}

	return s.wishlistRepo.Create(wishlist)
}

// GetWishlists retrieves all wishlists for a workspace
func (s *WishlistService) GetWishlists(workspaceID int32) ([]*domain.Wishlist, error) {
	return s.wishlistRepo.GetAllByWorkspace(workspaceID)
}

// GetWishlistByID retrieves a wishlist by ID within a workspace
func (s *WishlistService) GetWishlistByID(workspaceID int32, id int32) (*domain.Wishlist, error) {
	return s.wishlistRepo.GetByID(workspaceID, id)
}

// UpdateWishlistInput contains input for updating a wishlist
type UpdateWishlistInput struct {
	Name string
}

// UpdateWishlist updates a wishlist (rename)
func (s *WishlistService) UpdateWishlist(workspaceID int32, id int32, input UpdateWishlistInput) (*domain.Wishlist, error) {
	// Verify wishlist exists
	existing, err := s.wishlistRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Validate name
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrWishlistNameEmpty
	}
	if len(name) > 255 {
		return nil, domain.ErrWishlistNameTooLong
	}

	existing.Name = name

	return s.wishlistRepo.Update(existing)
}

// DeleteWishlist soft-deletes a wishlist
func (s *WishlistService) DeleteWishlist(workspaceID int32, id int32) error {
	// Verify wishlist exists before deleting
	_, err := s.wishlistRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	// NOTE: Item count check and cascade/move options will be added in Story 8-2
	// when wishlist_items table exists. For now, allow deletion without checking.

	return s.wishlistRepo.SoftDelete(workspaceID, id)
}
