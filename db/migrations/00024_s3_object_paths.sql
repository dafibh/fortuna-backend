-- +goose Up
-- +goose StatementBegin

-- Add image_path column to wishlist_items
ALTER TABLE wishlist_items ADD COLUMN image_path TEXT;

-- Extract path from URL: http://endpoint/bucket/{path} -> {path}
-- Pattern: removes protocol + host + bucket name, leaving just the object path
UPDATE wishlist_items
SET image_path = regexp_replace(image_url, '^https?://[^/]+/[^/]+/', '')
WHERE image_url IS NOT NULL AND image_url != '';

-- Add image_path column to wishlist_item_notes
ALTER TABLE wishlist_item_notes ADD COLUMN image_path TEXT;

UPDATE wishlist_item_notes
SET image_path = regexp_replace(image_url, '^https?://[^/]+/[^/]+/', '')
WHERE image_url IS NOT NULL AND image_url != '';

-- Add image_path column to wishlist_item_prices
ALTER TABLE wishlist_item_prices ADD COLUMN image_path TEXT;

UPDATE wishlist_item_prices
SET image_path = regexp_replace(image_url, '^https?://[^/]+/[^/]+/', '')
WHERE image_url IS NOT NULL AND image_url != '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop image_path columns
-- NOTE: This is a LOSSY rollback. The image_url column still contains the original
-- full URLs, so data is preserved. However, any NEW paths stored after migration
-- that don't have corresponding image_url values will be lost.
-- This is acceptable because:
-- 1. image_url retains pre-migration data
-- 2. Rollback is only for emergencies, not routine use
-- 3. A future migration (Story 11.5) will drop image_url after frontend migration
ALTER TABLE wishlist_items DROP COLUMN IF EXISTS image_path;
ALTER TABLE wishlist_item_notes DROP COLUMN IF EXISTS image_path;
ALTER TABLE wishlist_item_prices DROP COLUMN IF EXISTS image_path;

-- +goose StatementEnd
