-- +goose Up
-- +goose StatementBegin
CREATE TABLE wishlist_item_notes (
    id SERIAL PRIMARY KEY,
    item_id INT NOT NULL REFERENCES wishlist_items(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_wishlist_item_notes_item_created
    ON wishlist_item_notes(item_id, created_at DESC)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wishlist_item_notes_item_created;
DROP TABLE IF EXISTS wishlist_item_notes;
-- +goose StatementEnd
