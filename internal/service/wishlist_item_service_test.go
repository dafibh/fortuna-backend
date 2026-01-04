package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
)

func TestCreateItem_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title:       "Test Item",
		Description: strPtr("A test item"),
	}

	item, err := svc.CreateItem(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Title != "Test Item" {
		t.Errorf("expected title 'Test Item', got '%s'", item.Title)
	}
	if item.WishlistID != 1 {
		t.Errorf("expected wishlistID 1, got %d", item.WishlistID)
	}
}

func TestCreateItem_TrimsTitle(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title: "  Trimmed Title  ",
	}

	item, err := svc.CreateItem(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Title != "Trimmed Title" {
		t.Errorf("expected trimmed title 'Trimmed Title', got '%s'", item.Title)
	}
}

func TestCreateItem_EmptyTitle(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title: "",
	}

	_, err := svc.CreateItem(1, 1, input)
	if err != domain.ErrWishlistItemTitleEmpty {
		t.Errorf("expected ErrWishlistItemTitleEmpty, got %v", err)
	}
}

func TestCreateItem_WhitespaceOnlyTitle(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title: "   ",
	}

	_, err := svc.CreateItem(1, 1, input)
	if err != domain.ErrWishlistItemTitleEmpty {
		t.Errorf("expected ErrWishlistItemTitleEmpty, got %v", err)
	}
}

func TestCreateItem_TitleTooLong(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	longTitle := make([]byte, 256)
	for i := range longTitle {
		longTitle[i] = 'a'
	}
	input := CreateWishlistItemInput{
		Title: string(longTitle),
	}

	_, err := svc.CreateItem(1, 1, input)
	if err != domain.ErrWishlistItemTitleLong {
		t.Errorf("expected ErrWishlistItemTitleLong, got %v", err)
	}
}

func TestCreateItem_InvalidURL(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title:        "Test Item",
		ExternalLink: strPtr("not-a-valid-url"),
	}

	_, err := svc.CreateItem(1, 1, input)
	if err != domain.ErrWishlistItemInvalidURL {
		t.Errorf("expected ErrWishlistItemInvalidURL, got %v", err)
	}
}

func TestCreateItem_ValidURL(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title:        "Test Item",
		ExternalLink: strPtr("https://example.com/product"),
	}

	item, err := svc.CreateItem(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *item.ExternalLink != "https://example.com/product" {
		t.Errorf("expected URL 'https://example.com/product', got '%s'", *item.ExternalLink)
	}
}

func TestCreateItem_WithImagePath(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title:     "Test Item",
		ImagePath: strPtr("1/wishlist_items/1/uuid_display.jpg"),
	}

	item, err := svc.CreateItem(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *item.ImagePath != "1/wishlist_items/1/uuid_display.jpg" {
		t.Errorf("expected imagePath '1/wishlist_items/1/uuid_display.jpg', got '%s'", *item.ImagePath)
	}
}

func TestCreateItem_WishlistNotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := CreateWishlistItemInput{
		Title: "Test Item",
	}

	_, err := svc.CreateItem(1, 999, input)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("expected ErrWishlistNotFound, got %v", err)
	}
}

func TestGetItemByID_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Test Item",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	item, err := svc.GetItemByID(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Title != "Test Item" {
		t.Errorf("expected title 'Test Item', got '%s'", item.Title)
	}
}

func TestGetItemByID_NotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	_, err := svc.GetItemByID(1, 999)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestGetItemsByWishlist_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Item 1"})
	itemRepo.AddItem(&domain.WishlistItem{ID: 2, WishlistID: 1, Title: "Item 2"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	items, err := svc.GetItemsByWishlist(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestGetItemsByWishlist_EmptyList(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "My Wishlist"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	items, err := svc.GetItemsByWishlist(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestGetItemsByWishlist_WishlistNotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	_, err := svc.GetItemsByWishlist(1, 999)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("expected ErrWishlistNotFound, got %v", err)
	}
}

func TestUpdateItem_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Original Title",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := UpdateWishlistItemInput{
		Title:       "Updated Title",
		Description: strPtr("Updated description"),
	}

	item, err := svc.UpdateItem(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", item.Title)
	}
}

func TestUpdateItem_EmptyTitle(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Original Title",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := UpdateWishlistItemInput{
		Title: "",
	}

	_, err := svc.UpdateItem(1, 1, input)
	if err != domain.ErrWishlistItemTitleEmpty {
		t.Errorf("expected ErrWishlistItemTitleEmpty, got %v", err)
	}
}

func TestUpdateItem_NotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := UpdateWishlistItemInput{
		Title: "New Title",
	}

	_, err := svc.UpdateItem(1, 999, input)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestUpdateItem_InvalidURL(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Original Title",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	input := UpdateWishlistItemInput{
		Title:        "Updated Title",
		ExternalLink: strPtr("invalid-url"),
	}

	_, err := svc.UpdateItem(1, 1, input)
	if err != domain.ErrWishlistItemInvalidURL {
		t.Errorf("expected ErrWishlistItemInvalidURL, got %v", err)
	}
}

func TestMoveItem_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "Source"})
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 2, WorkspaceID: 1, Name: "Target"})

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item to Move",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	item, err := svc.MoveItem(1, 1, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.WishlistID != 2 {
		t.Errorf("expected wishlistID 2, got %d", item.WishlistID)
	}
}

func TestMoveItem_SameWishlist(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "Same"})

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	item, err := svc.MoveItem(1, 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should return same item without calling move
	if item.WishlistID != 1 {
		t.Errorf("expected wishlistID 1, got %d", item.WishlistID)
	}
}

func TestMoveItem_ItemNotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 2, WorkspaceID: 1, Name: "Target"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	_, err := svc.MoveItem(1, 999, 2)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestMoveItem_TargetWishlistNotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistRepo.AddWishlist(&domain.Wishlist{ID: 1, WorkspaceID: 1, Name: "Source"})

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	_, err := svc.MoveItem(1, 1, 999)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("expected ErrWishlistNotFound, got %v", err)
	}
}

func TestDeleteItem_Success(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item to Delete",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	err := svc.DeleteItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify item is deleted
	_, err = svc.GetItemByID(1, 1)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound after delete, got %v", err)
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	err := svc.DeleteItem(1, 999)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestGetWishlistThumbnail_WithImage(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	imagePath := "1/wishlist_items/1/uuid_display.jpg"
	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item with Image",
		ImagePath:  &imagePath,
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	path, err := svc.GetWishlistThumbnail(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path == nil || *path != imagePath {
		t.Errorf("expected path '%s', got '%v'", imagePath, path)
	}
}

func TestGetWishlistThumbnail_NoImage(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{
		ID:         1,
		WishlistID: 1,
		Title:      "Item without Image",
	})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	url, err := svc.GetWishlistThumbnail(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != nil {
		t.Errorf("expected nil URL, got '%s'", *url)
	}
}

func TestGetWishlistItemCount_WithItems(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Item 1"})
	itemRepo.AddItem(&domain.WishlistItem{ID: 2, WishlistID: 1, Title: "Item 2"})
	itemRepo.AddItem(&domain.WishlistItem{ID: 3, WishlistID: 1, Title: "Item 3"})

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	count, err := svc.GetWishlistItemCount(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestGetWishlistItemCount_Empty(t *testing.T) {
	itemRepo := testutil.NewMockWishlistItemRepository()
	wishlistRepo := testutil.NewMockWishlistRepository()

	svc := NewWishlistItemService(itemRepo, wishlistRepo)

	count, err := svc.GetWishlistItemCount(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
