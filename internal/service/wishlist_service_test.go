package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
)

// CreateWishlist tests

func TestCreateWishlist_Success(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	input := CreateWishlistInput{
		Name: "Birthday Gifts",
	}

	wishlist, err := wishlistService.CreateWishlist(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wishlist.Name != "Birthday Gifts" {
		t.Errorf("Expected name 'Birthday Gifts', got %s", wishlist.Name)
	}

	if wishlist.WorkspaceID != workspaceID {
		t.Errorf("Expected workspace ID %d, got %d", workspaceID, wishlist.WorkspaceID)
	}
}

func TestCreateWishlist_TrimsName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	input := CreateWishlistInput{
		Name: "  Holiday Wishlist  ",
	}

	wishlist, err := wishlistService.CreateWishlist(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wishlist.Name != "Holiday Wishlist" {
		t.Errorf("Expected trimmed name 'Holiday Wishlist', got '%s'", wishlist.Name)
	}
}

func TestCreateWishlist_EmptyName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	input := CreateWishlistInput{
		Name: "",
	}

	_, err := wishlistService.CreateWishlist(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for empty name, got nil")
	}

	if err != domain.ErrWishlistNameEmpty {
		t.Errorf("Expected ErrWishlistNameEmpty, got %v", err)
	}
}

func TestCreateWishlist_WhitespaceOnlyName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	input := CreateWishlistInput{
		Name: "   ",
	}

	_, err := wishlistService.CreateWishlist(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for whitespace-only name, got nil")
	}

	if err != domain.ErrWishlistNameEmpty {
		t.Errorf("Expected ErrWishlistNameEmpty, got %v", err)
	}
}

func TestCreateWishlist_NameTooLong(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	// Create a name that's 256 characters long
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "A"
	}
	input := CreateWishlistInput{
		Name: longName,
	}

	_, err := wishlistService.CreateWishlist(workspaceID, input)
	if err == nil {
		t.Fatal("Expected error for name > 255 characters, got nil")
	}

	if err != domain.ErrWishlistNameTooLong {
		t.Errorf("Expected ErrWishlistNameTooLong, got %v", err)
	}
}

func TestCreateWishlist_NameExactly255Characters(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	// Create a name that's exactly 255 characters
	name255 := ""
	for i := 0; i < 255; i++ {
		name255 += "A"
	}
	input := CreateWishlistInput{
		Name: name255,
	}

	wishlist, err := wishlistService.CreateWishlist(workspaceID, input)
	if err != nil {
		t.Fatalf("Expected no error for name = 255 characters, got %v", err)
	}

	if len(wishlist.Name) != 255 {
		t.Errorf("Expected name length 255, got %d", len(wishlist.Name))
	}
}

func TestCreateWishlist_DuplicateName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	// Create first wishlist
	input1 := CreateWishlistInput{Name: "My Wishlist"}
	_, err := wishlistService.CreateWishlist(workspaceID, input1)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Try to create second wishlist with same name
	input2 := CreateWishlistInput{Name: "My Wishlist"}
	_, err = wishlistService.CreateWishlist(workspaceID, input2)
	if err != domain.ErrWishlistNameExists {
		t.Errorf("Expected ErrWishlistNameExists, got %v", err)
	}
}

// GetWishlists tests

func TestGetWishlists_Success(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	// Add some wishlists
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Wishlist 1",
	})
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Wishlist 2",
	})

	wishlists, err := wishlistService.GetWishlists(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(wishlists) != 2 {
		t.Errorf("Expected 2 wishlists, got %d", len(wishlists))
	}
}

func TestGetWishlists_EmptyList(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	wishlists, err := wishlistService.GetWishlists(workspaceID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(wishlists) != 0 {
		t.Errorf("Expected 0 wishlists, got %d", len(wishlists))
	}
}

func TestGetWishlists_WorkspaceIsolation(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	// Add wishlist to workspace 1
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Wishlist WS1",
	})

	// Add wishlist to workspace 2
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          2,
		WorkspaceID: 2,
		Name:        "Wishlist WS2",
	})

	// Query workspace 1 - should only see 1 wishlist
	wishlists1, err := wishlistService.GetWishlists(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(wishlists1) != 1 {
		t.Errorf("Expected 1 wishlist for workspace 1, got %d", len(wishlists1))
	}
	if wishlists1[0].Name != "Wishlist WS1" {
		t.Errorf("Expected 'Wishlist WS1', got %s", wishlists1[0].Name)
	}

	// Query workspace 2 - should only see 1 wishlist
	wishlists2, err := wishlistService.GetWishlists(2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(wishlists2) != 1 {
		t.Errorf("Expected 1 wishlist for workspace 2, got %d", len(wishlists2))
	}
	if wishlists2[0].Name != "Wishlist WS2" {
		t.Errorf("Expected 'Wishlist WS2', got %s", wishlists2[0].Name)
	}
}

// GetWishlistByID tests

func TestGetWishlistByID_Success(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistID := int32(1)

	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          wishlistID,
		WorkspaceID: workspaceID,
		Name:        "Test Wishlist",
	})

	wishlist, err := wishlistService.GetWishlistByID(workspaceID, wishlistID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wishlist.Name != "Test Wishlist" {
		t.Errorf("Expected name 'Test Wishlist', got %s", wishlist.Name)
	}
}

func TestGetWishlistByID_NotFound(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	_, err := wishlistService.GetWishlistByID(workspaceID, 999)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound, got %v", err)
	}
}

func TestGetWishlistByID_WrongWorkspace(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	// Wishlist belongs to workspace 1
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Wishlist",
	})

	// Try to get it from workspace 2
	_, err := wishlistService.GetWishlistByID(2, 1)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound for wrong workspace, got %v", err)
	}
}

// UpdateWishlist tests

func TestUpdateWishlist_Success(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	input := UpdateWishlistInput{
		Name: "New Name",
	}

	wishlist, err := wishlistService.UpdateWishlist(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wishlist.Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", wishlist.Name)
	}
}

func TestUpdateWishlist_TrimsName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	input := UpdateWishlistInput{
		Name: "  New Name  ",
	}

	wishlist, err := wishlistService.UpdateWishlist(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wishlist.Name != "New Name" {
		t.Errorf("Expected trimmed name 'New Name', got '%s'", wishlist.Name)
	}
}

func TestUpdateWishlist_EmptyName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Old Name",
	})

	input := UpdateWishlistInput{
		Name: "",
	}

	_, err := wishlistService.UpdateWishlist(workspaceID, 1, input)
	if err != domain.ErrWishlistNameEmpty {
		t.Errorf("Expected ErrWishlistNameEmpty, got %v", err)
	}
}

func TestUpdateWishlist_NameTooLong(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Wishlist",
	})

	// Create a name that's 256 characters long
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "A"
	}
	input := UpdateWishlistInput{
		Name: longName,
	}

	_, err := wishlistService.UpdateWishlist(workspaceID, 1, input)
	if err == nil {
		t.Fatal("Expected error for name > 255 characters, got nil")
	}

	if err != domain.ErrWishlistNameTooLong {
		t.Errorf("Expected ErrWishlistNameTooLong, got %v", err)
	}
}

func TestUpdateWishlist_NotFound(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	input := UpdateWishlistInput{
		Name: "New Name",
	}

	_, err := wishlistService.UpdateWishlist(workspaceID, 999, input)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound, got %v", err)
	}
}

func TestUpdateWishlist_WrongWorkspace(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	// Wishlist belongs to workspace 1
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Wishlist",
	})

	input := UpdateWishlistInput{
		Name: "New Name",
	}

	// Try to update it from workspace 2
	_, err := wishlistService.UpdateWishlist(2, 1, input)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound for wrong workspace, got %v", err)
	}
}

func TestUpdateWishlist_DuplicateName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	// Add two wishlists
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Wishlist 1",
	})
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          2,
		WorkspaceID: workspaceID,
		Name:        "Wishlist 2",
	})

	// Try to rename wishlist 2 to wishlist 1's name
	input := UpdateWishlistInput{
		Name: "Wishlist 1",
	}

	_, err := wishlistService.UpdateWishlist(workspaceID, 2, input)
	if err != domain.ErrWishlistNameExists {
		t.Errorf("Expected ErrWishlistNameExists, got %v", err)
	}
}

func TestUpdateWishlist_SameName(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "My Wishlist",
	})

	// Update to the same name - should succeed
	input := UpdateWishlistInput{
		Name: "My Wishlist",
	}

	wishlist, err := wishlistService.UpdateWishlist(workspaceID, 1, input)
	if err != nil {
		t.Fatalf("Expected no error when updating to same name, got %v", err)
	}

	if wishlist.Name != "My Wishlist" {
		t.Errorf("Expected name 'My Wishlist', got %s", wishlist.Name)
	}
}

// DeleteWishlist tests

func TestDeleteWishlist_Success(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Wishlist",
	})

	err := wishlistService.DeleteWishlist(workspaceID, 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify wishlist is soft-deleted (not found when querying)
	_, err = wishlistService.GetWishlistByID(workspaceID, 1)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound after soft delete, got %v", err)
	}
}

func TestDeleteWishlist_NotFound(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)

	err := wishlistService.DeleteWishlist(workspaceID, 999)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound, got %v", err)
	}
}

func TestDeleteWishlist_WrongWorkspace(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	// Wishlist belongs to workspace 1
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: 1,
		Name:        "Test Wishlist",
	})

	// Try to delete it from workspace 2
	err := wishlistService.DeleteWishlist(2, 1)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound for wrong workspace, got %v", err)
	}
}

func TestDeleteWishlist_AlreadyDeleted(t *testing.T) {
	wishlistRepo := testutil.NewMockWishlistRepository()
	wishlistService := NewWishlistService(wishlistRepo)

	workspaceID := int32(1)
	wishlistRepo.AddWishlist(&domain.Wishlist{
		ID:          1,
		WorkspaceID: workspaceID,
		Name:        "Test Wishlist",
	})

	// First delete should succeed
	err := wishlistService.DeleteWishlist(workspaceID, 1)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Second delete should fail (already deleted)
	err = wishlistService.DeleteWishlist(workspaceID, 1)
	if err != domain.ErrWishlistNotFound {
		t.Errorf("Expected ErrWishlistNotFound for already deleted wishlist, got %v", err)
	}
}
