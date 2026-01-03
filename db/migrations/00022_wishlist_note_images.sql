-- +goose Up
-- +goose StatementBegin
ALTER TABLE wishlist_item_notes
ADD COLUMN image_url VARCHAR(2048);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE wishlist_item_notes
DROP COLUMN IF EXISTS image_url;
-- +goose StatementEnd
