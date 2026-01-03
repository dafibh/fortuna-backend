package postgres

import (
	"context"

	"github.com/dafibh/fortuna/fortuna-backend/db/sqlc"
	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WishlistNoteRepository implements domain.WishlistNoteRepository using PostgreSQL
type WishlistNoteRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewWishlistNoteRepository creates a new WishlistNoteRepository
func NewWishlistNoteRepository(pool *pgxpool.Pool) *WishlistNoteRepository {
	return &WishlistNoteRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Create creates a new note
func (r *WishlistNoteRepository) Create(note *domain.WishlistItemNote) (*domain.WishlistItemNote, error) {
	ctx := context.Background()
	created, err := r.queries.CreateWishlistItemNote(ctx, sqlc.CreateWishlistItemNoteParams{
		ItemID:   note.ItemID,
		Content:  note.Content,
		ImageUrl: stringPtrToPgText(note.ImageURL),
	})
	if err != nil {
		return nil, err
	}
	return sqlcNoteToDomain(created), nil
}

// GetByID retrieves a note by its ID within a workspace
func (r *WishlistNoteRepository) GetByID(workspaceID int32, id int32) (*domain.WishlistItemNote, error) {
	ctx := context.Background()
	note, err := r.queries.GetWishlistItemNoteByID(ctx, sqlc.GetWishlistItemNoteByIDParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNoteNotFound
		}
		return nil, err
	}
	return sqlcNoteToDomain(note), nil
}

// ListByItem retrieves all notes for an item
func (r *WishlistNoteRepository) ListByItem(workspaceID int32, itemID int32, sortAsc bool) ([]*domain.WishlistItemNote, error) {
	ctx := context.Background()

	var notes []sqlc.WishlistItemNote
	var err error

	if sortAsc {
		notes, err = r.queries.ListNotesByItemAsc(ctx, sqlc.ListNotesByItemAscParams{
			ItemID:      itemID,
			WorkspaceID: workspaceID,
		})
	} else {
		notes, err = r.queries.ListNotesByItemDesc(ctx, sqlc.ListNotesByItemDescParams{
			ItemID:      itemID,
			WorkspaceID: workspaceID,
		})
	}

	if err != nil {
		return nil, err
	}

	result := make([]*domain.WishlistItemNote, len(notes))
	for i, note := range notes {
		result[i] = sqlcNoteToDomain(note)
	}
	return result, nil
}

// CountByItem counts notes for an item
func (r *WishlistNoteRepository) CountByItem(workspaceID int32, itemID int32) (int64, error) {
	ctx := context.Background()
	return r.queries.CountNotesByItem(ctx, sqlc.CountNotesByItemParams{
		ItemID:      itemID,
		WorkspaceID: workspaceID,
	})
}

// Update updates a note's content and image
func (r *WishlistNoteRepository) Update(workspaceID int32, id int32, content string, imageURL *string) (*domain.WishlistItemNote, error) {
	ctx := context.Background()
	updated, err := r.queries.UpdateWishlistItemNote(ctx, sqlc.UpdateWishlistItemNoteParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Content:     content,
		ImageUrl:    stringPtrToPgText(imageURL),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNoteNotFound
		}
		return nil, err
	}
	return sqlcNoteToDomain(updated), nil
}

// Delete soft-deletes a note
func (r *WishlistNoteRepository) Delete(workspaceID int32, id int32) error {
	ctx := context.Background()
	return r.queries.DeleteWishlistItemNote(ctx, sqlc.DeleteWishlistItemNoteParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

// Helper function to convert sqlc type to domain type
func sqlcNoteToDomain(note sqlc.WishlistItemNote) *domain.WishlistItemNote {
	var imageURL *string
	if note.ImageUrl.Valid {
		imageURL = &note.ImageUrl.String
	}
	return &domain.WishlistItemNote{
		ID:        note.ID,
		ItemID:    note.ItemID,
		Content:   note.Content,
		ImageURL:  imageURL,
		CreatedAt: note.CreatedAt.Time,
		UpdatedAt: note.UpdatedAt.Time,
	}
}
