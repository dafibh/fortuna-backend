-- +goose Up
-- +goose StatementBegin
CREATE TABLE wishlist_item_prices (
    id SERIAL PRIMARY KEY,
    item_id INTEGER NOT NULL REFERENCES wishlist_items(id) ON DELETE CASCADE,
    platform_name VARCHAR(100) NOT NULL,
    price NUMERIC(12,2) NOT NULL CHECK (price > 0),
    price_date DATE NOT NULL DEFAULT CURRENT_DATE,
    image_url VARCHAR(2048),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- NO updated_at or deleted_at: entries are immutable
);

CREATE INDEX idx_wishlist_item_prices_item_platform_date
    ON wishlist_item_prices(item_id, platform_name, price_date DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS wishlist_item_prices;
-- +goose StatementEnd
