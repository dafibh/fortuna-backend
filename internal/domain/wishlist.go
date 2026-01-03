package domain

import (
	"errors"
	"time"
)

var (
	ErrWishlistNotFound    = errors.New("wishlist not found")
	ErrWishlistNameExists  = errors.New("wishlist with this name already exists")
	ErrWishlistNameEmpty   = errors.New("wishlist name is required")
	ErrWishlistNameTooLong = errors.New("wishlist name must be 255 characters or less")
	ErrWishlistHasItems    = errors.New("wishlist has items")
)

type Wishlist struct {
	ID          int32      `json:"id"`
	WorkspaceID int32      `json:"workspaceId"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// WishlistWithStats includes item count for list views
type WishlistWithStats struct {
	Wishlist
	ItemCount    int     `json:"itemCount"`
	ThumbnailURL *string `json:"thumbnailUrl"`
}

func (w *Wishlist) Validate() error {
	if w.Name == "" {
		return ErrWishlistNameEmpty
	}
	if len(w.Name) > 255 {
		return ErrWishlistNameTooLong
	}
	return nil
}

type WishlistRepository interface {
	Create(wishlist *Wishlist) (*Wishlist, error)
	GetByID(workspaceID int32, id int32) (*Wishlist, error)
	GetByName(workspaceID int32, name string) (*Wishlist, error)
	GetAllByWorkspace(workspaceID int32) ([]*Wishlist, error)
	Update(wishlist *Wishlist) (*Wishlist, error)
	SoftDelete(workspaceID int32, id int32) error
	// CountItems and GetAllWithStats deferred to Story 8-2 when wishlist_items table exists
}
