package service

import (
	"testing"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
)

func TestCreateNote_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	input := CreateNoteInput{
		Content: "This is a research note",
	}

	note, err := svc.CreateNote(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note.Content != "This is a research note" {
		t.Errorf("expected content 'This is a research note', got '%s'", note.Content)
	}
	if note.ItemID != 1 {
		t.Errorf("expected itemID 1, got %d", note.ItemID)
	}
}

func TestCreateNote_TrimsContent(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	input := CreateNoteInput{
		Content: "  Trimmed content  ",
	}

	note, err := svc.CreateNote(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note.Content != "Trimmed content" {
		t.Errorf("expected trimmed content 'Trimmed content', got '%s'", note.Content)
	}
}

func TestCreateNote_EmptyContent(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	input := CreateNoteInput{
		Content: "",
	}

	_, err := svc.CreateNote(1, 1, input)
	if err != domain.ErrNoteContentEmpty {
		t.Errorf("expected ErrNoteContentEmpty, got %v", err)
	}
}

func TestCreateNote_WhitespaceOnlyContent(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	input := CreateNoteInput{
		Content: "   ",
	}

	_, err := svc.CreateNote(1, 1, input)
	if err != domain.ErrNoteContentEmpty {
		t.Errorf("expected ErrNoteContentEmpty, got %v", err)
	}
}

func TestCreateNote_ItemNotFound(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	input := CreateNoteInput{
		Content: "Test note",
	}

	_, err := svc.CreateNote(1, 999, input)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestGetNoteByID_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "Test note"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	note, err := svc.GetNoteByID(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note.Content != "Test note" {
		t.Errorf("expected content 'Test note', got '%s'", note.Content)
	}
}

func TestGetNoteByID_NotFound(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	_, err := svc.GetNoteByID(1, 999)
	if err != domain.ErrNoteNotFound {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestListNotesByItem_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "First note"})
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 2, ItemID: 1, Content: "Second note"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	notes, err := svc.ListNotesByItem(1, 1, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestListNotesByItem_ItemNotFound(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	_, err := svc.ListNotesByItem(1, 999, false)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestCountNotesByItem_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "First note"})
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 2, ItemID: 1, Content: "Second note"})
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 3, ItemID: 1, Content: "Third note"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	count, err := svc.CountNotesByItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestUpdateNote_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "Original content"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	note, err := svc.UpdateNote(1, 1, "Updated content", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", note.Content)
	}
}

func TestUpdateNote_EmptyContent(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "Original content"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	_, err := svc.UpdateNote(1, 1, "", nil)
	if err != domain.ErrNoteContentEmpty {
		t.Errorf("expected ErrNoteContentEmpty, got %v", err)
	}
}

func TestUpdateNote_NotFound(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	_, err := svc.UpdateNote(1, 999, "New content", nil)
	if err != domain.ErrNoteNotFound {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestDeleteNote_Success(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	noteRepo.AddNote(&domain.WishlistItemNote{ID: 1, ItemID: 1, Content: "Test note"})

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	err := svc.DeleteNote(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify note is deleted
	_, err = svc.GetNoteByID(1, 1)
	if err != domain.ErrNoteNotFound {
		t.Errorf("expected ErrNoteNotFound after delete, got %v", err)
	}
}

func TestDeleteNote_NotFound(t *testing.T) {
	noteRepo := testutil.NewMockWishlistNoteRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistNoteService(noteRepo, itemRepo)

	err := svc.DeleteNote(1, 999)
	if err != domain.ErrNoteNotFound {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}
