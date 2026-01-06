-- name: CreateWishlistItem :one
INSERT INTO wishlist_items (wishlist_id, title, description, external_link, image_path)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetWishlistItemByID :one
SELECT wi.* FROM wishlist_items wi
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wi.id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: ListWishlistItems :many
SELECT wi.* FROM wishlist_items wi
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wi.wishlist_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wi.created_at DESC;

-- name: UpdateWishlistItem :one
UPDATE wishlist_items wi
SET title = $3, description = $4, external_link = $5, image_path = $6, updated_at = NOW()
FROM wishlists w
WHERE wi.id = $1 AND w.id = wi.wishlist_id AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING wi.*;

-- name: MoveWishlistItem :one
UPDATE wishlist_items wi
SET wishlist_id = $3, updated_at = NOW()
FROM wishlists w
WHERE wi.id = $1 AND w.id = wi.wishlist_id AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING wi.*;

-- name: DeleteWishlistItem :exec
UPDATE wishlist_items wi
SET deleted_at = NOW()
FROM wishlists w
WHERE wi.id = $1 AND w.id = wi.wishlist_id AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: GetFirstItemImage :one
SELECT wi.image_path FROM wishlist_items wi
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wi.wishlist_id = $1 AND w.workspace_id = $2 AND wi.image_path IS NOT NULL AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wi.created_at ASC
LIMIT 1;

-- name: CountWishlistItems :one
SELECT COUNT(*) FROM wishlist_items wi
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wi.wishlist_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: ListWishlistItemsWithStats :many
SELECT
    wi.*,
    COALESCE(
        (
            SELECT MIN(current_prices.price)::TEXT
            FROM (
                SELECT DISTINCT ON (wip.platform_name) wip.price
                FROM wishlist_item_prices wip
                WHERE wip.item_id = wi.id
                ORDER BY wip.platform_name, wip.price_date DESC, wip.created_at DESC
            ) AS current_prices
        ),
        ''
    ) AS best_price,
    0::int AS note_count  -- Placeholder for Story 8-5
FROM wishlist_items wi
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wi.wishlist_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wi.created_at DESC;
