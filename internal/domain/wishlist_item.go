package domain

import (
	"errors"
	"net/url"
	"time"
)

var (
	ErrWishlistItemNotFound        = errors.New("wishlist item not found")
	ErrWishlistItemTitleEmpty      = errors.New("wishlist item title is required")
	ErrWishlistItemTitleLong       = errors.New("wishlist item title must be 255 characters or less")
	ErrWishlistItemInvalidURL      = errors.New("external link must be a valid URL")
	ErrWishlistItemInvalidImageURL = errors.New("image URL must be a valid URL")
)

type WishlistItem struct {
	ID           int32      `json:"id"`
	WishlistID   int32      `json:"wishlistId"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	ExternalLink *string    `json:"externalLink,omitempty"`
	ImageURL     *string    `json:"imageUrl,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

// WishlistItemWithStats includes best price and note count for list views
type WishlistItemWithStats struct {
	WishlistItem
	BestPrice *string `json:"bestPrice,omitempty"`
	NoteCount int     `json:"noteCount"`
}

func (item *WishlistItem) Validate() error {
	if item.Title == "" {
		return ErrWishlistItemTitleEmpty
	}
	if len(item.Title) > 255 {
		return ErrWishlistItemTitleLong
	}
	if item.ExternalLink != nil && *item.ExternalLink != "" {
		if _, err := url.ParseRequestURI(*item.ExternalLink); err != nil {
			return ErrWishlistItemInvalidURL
		}
	}
	if item.ImageURL != nil && *item.ImageURL != "" {
		if _, err := url.ParseRequestURI(*item.ImageURL); err != nil {
			return ErrWishlistItemInvalidImageURL
		}
	}
	return nil
}

type WishlistItemRepository interface {
	Create(item *WishlistItem) (*WishlistItem, error)
	GetByID(workspaceID int32, id int32) (*WishlistItem, error)
	GetAllByWishlist(workspaceID int32, wishlistID int32) ([]*WishlistItem, error)
	Update(workspaceID int32, item *WishlistItem) (*WishlistItem, error)
	Move(workspaceID int32, itemID int32, targetWishlistID int32) (*WishlistItem, error)
	SoftDelete(workspaceID int32, id int32) error
	GetFirstItemImage(workspaceID int32, wishlistID int32) (*string, error)
	CountByWishlist(workspaceID int32, wishlistID int32) (int64, error)
}
