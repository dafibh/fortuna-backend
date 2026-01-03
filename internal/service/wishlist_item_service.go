package service

import (
	"net/url"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
)

// WishlistItemService handles wishlist item business logic
type WishlistItemService struct {
	itemRepo     domain.WishlistItemRepository
	wishlistRepo domain.WishlistRepository
}

// NewWishlistItemService creates a new WishlistItemService
func NewWishlistItemService(itemRepo domain.WishlistItemRepository, wishlistRepo domain.WishlistRepository) *WishlistItemService {
	return &WishlistItemService{
		itemRepo:     itemRepo,
		wishlistRepo: wishlistRepo,
	}
}

// CreateWishlistItemInput contains input for creating a wishlist item
type CreateWishlistItemInput struct {
	Title        string
	Description  *string
	ExternalLink *string
	ImageURL     *string
}

// CreateItem creates a new item in a wishlist
func (s *WishlistItemService) CreateItem(workspaceID int32, wishlistID int32, input CreateWishlistItemInput) (*domain.WishlistItem, error) {
	// Verify wishlist exists and belongs to workspace
	_, err := s.wishlistRepo.GetByID(workspaceID, wishlistID)
	if err != nil {
		return nil, err
	}

	// Validate title
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, domain.ErrWishlistItemTitleEmpty
	}
	if len(title) > 255 {
		return nil, domain.ErrWishlistItemTitleLong
	}

	// Validate URLs if provided
	if input.ExternalLink != nil && *input.ExternalLink != "" {
		link := strings.TrimSpace(*input.ExternalLink)
		if _, err := url.ParseRequestURI(link); err != nil {
			return nil, domain.ErrWishlistItemInvalidURL
		}
		input.ExternalLink = &link
	}
	if input.ImageURL != nil && *input.ImageURL != "" {
		imageURL := strings.TrimSpace(*input.ImageURL)
		if _, err := url.ParseRequestURI(imageURL); err != nil {
			return nil, domain.ErrWishlistItemInvalidImageURL
		}
		input.ImageURL = &imageURL
	}

	item := &domain.WishlistItem{
		WishlistID:   wishlistID,
		Title:        title,
		Description:  input.Description,
		ExternalLink: input.ExternalLink,
		ImageURL:     input.ImageURL,
	}

	return s.itemRepo.Create(item)
}

// GetItemByID retrieves a wishlist item by ID
func (s *WishlistItemService) GetItemByID(workspaceID int32, id int32) (*domain.WishlistItem, error) {
	return s.itemRepo.GetByID(workspaceID, id)
}

// GetItemsByWishlist retrieves all items in a wishlist
func (s *WishlistItemService) GetItemsByWishlist(workspaceID int32, wishlistID int32) ([]*domain.WishlistItem, error) {
	// Verify wishlist exists and belongs to workspace
	_, err := s.wishlistRepo.GetByID(workspaceID, wishlistID)
	if err != nil {
		return nil, err
	}

	return s.itemRepo.GetAllByWishlist(workspaceID, wishlistID)
}

// GetItemsByWishlistWithStats retrieves all items with best price and note count
func (s *WishlistItemService) GetItemsByWishlistWithStats(workspaceID int32, wishlistID int32) ([]*domain.WishlistItemWithStats, error) {
	// Verify wishlist exists and belongs to workspace
	_, err := s.wishlistRepo.GetByID(workspaceID, wishlistID)
	if err != nil {
		return nil, err
	}

	return s.itemRepo.GetAllByWishlistWithStats(workspaceID, wishlistID)
}

// UpdateWishlistItemInput contains input for updating a wishlist item
type UpdateWishlistItemInput struct {
	Title        string
	Description  *string
	ExternalLink *string
	ImageURL     *string
}

// UpdateItem updates a wishlist item
func (s *WishlistItemService) UpdateItem(workspaceID int32, id int32, input UpdateWishlistItemInput) (*domain.WishlistItem, error) {
	// Verify item exists
	existing, err := s.itemRepo.GetByID(workspaceID, id)
	if err != nil {
		return nil, err
	}

	// Validate title
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, domain.ErrWishlistItemTitleEmpty
	}
	if len(title) > 255 {
		return nil, domain.ErrWishlistItemTitleLong
	}

	// Validate URLs if provided
	if input.ExternalLink != nil && *input.ExternalLink != "" {
		link := strings.TrimSpace(*input.ExternalLink)
		if _, err := url.ParseRequestURI(link); err != nil {
			return nil, domain.ErrWishlistItemInvalidURL
		}
		input.ExternalLink = &link
	}
	if input.ImageURL != nil && *input.ImageURL != "" {
		imageURL := strings.TrimSpace(*input.ImageURL)
		if _, err := url.ParseRequestURI(imageURL); err != nil {
			return nil, domain.ErrWishlistItemInvalidImageURL
		}
		input.ImageURL = &imageURL
	}

	existing.Title = title
	existing.Description = input.Description
	existing.ExternalLink = input.ExternalLink
	existing.ImageURL = input.ImageURL

	return s.itemRepo.Update(workspaceID, existing)
}

// MoveItem moves an item to a different wishlist
func (s *WishlistItemService) MoveItem(workspaceID int32, itemID int32, targetWishlistID int32) (*domain.WishlistItem, error) {
	// Verify item exists
	item, err := s.itemRepo.GetByID(workspaceID, itemID)
	if err != nil {
		return nil, err
	}

	// Verify target wishlist exists and belongs to workspace
	_, err = s.wishlistRepo.GetByID(workspaceID, targetWishlistID)
	if err != nil {
		return nil, err
	}

	// Don't move if already in target wishlist
	if item.WishlistID == targetWishlistID {
		return item, nil
	}

	return s.itemRepo.Move(workspaceID, itemID, targetWishlistID)
}

// DeleteItem soft-deletes a wishlist item
func (s *WishlistItemService) DeleteItem(workspaceID int32, id int32) error {
	// Verify item exists
	_, err := s.itemRepo.GetByID(workspaceID, id)
	if err != nil {
		return err
	}

	return s.itemRepo.SoftDelete(workspaceID, id)
}

// GetWishlistThumbnail gets the first item's image for a wishlist thumbnail
func (s *WishlistItemService) GetWishlistThumbnail(workspaceID int32, wishlistID int32) (*string, error) {
	return s.itemRepo.GetFirstItemImage(workspaceID, wishlistID)
}

// GetWishlistItemCount gets the count of items in a wishlist
func (s *WishlistItemService) GetWishlistItemCount(workspaceID int32, wishlistID int32) (int64, error) {
	return s.itemRepo.CountByWishlist(workspaceID, wishlistID)
}
