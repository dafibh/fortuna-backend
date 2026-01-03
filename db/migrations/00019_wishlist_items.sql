-- +goose Up
-- +goose StatementBegin
CREATE TABLE wishlist_items (
    id SERIAL PRIMARY KEY,
    wishlist_id INT NOT NULL REFERENCES wishlists(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    external_link VARCHAR(2048),
    image_url VARCHAR(2048),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_wishlist_items_wishlist
    ON wishlist_items(wishlist_id)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wishlist_items_wishlist;
DROP TABLE IF EXISTS wishlist_items;
-- +goose StatementEnd
