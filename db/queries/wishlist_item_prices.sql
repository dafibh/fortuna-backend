-- name: CreateWishlistItemPrice :one
INSERT INTO wishlist_item_prices (item_id, platform_name, price, price_date, image_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetWishlistItemPriceByID :one
SELECT wip.* FROM wishlist_item_prices wip
JOIN wishlist_items wi ON wi.id = wip.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wip.id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: ListPricesByItem :many
SELECT wip.* FROM wishlist_item_prices wip
JOIN wishlist_items wi ON wi.id = wip.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wip.item_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wip.platform_name, wip.price_date DESC, wip.created_at DESC;

-- name: GetCurrentPricesByItem :many
SELECT DISTINCT ON (wip.platform_name)
    wip.id, wip.item_id, wip.platform_name, wip.price, wip.price_date, wip.image_url, wip.created_at
FROM wishlist_item_prices wip
JOIN wishlist_items wi ON wi.id = wip.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wip.item_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wip.platform_name, wip.price_date DESC, wip.created_at DESC;

-- name: GetBestPriceForItem :one
SELECT MIN(current_prices.price)::TEXT as best_price
FROM (
    SELECT DISTINCT ON (wip.platform_name) wip.price
    FROM wishlist_item_prices wip
    JOIN wishlist_items wi ON wi.id = wip.item_id
    JOIN wishlists w ON w.id = wi.wishlist_id
    WHERE wip.item_id = $1 AND w.workspace_id = $2 AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
    ORDER BY wip.platform_name, wip.price_date DESC, wip.created_at DESC
) AS current_prices;

-- name: DeleteWishlistItemPrice :exec
DELETE FROM wishlist_item_prices wip
USING wishlist_items wi, wishlists w
WHERE wip.id = $1 AND wip.item_id = wi.id AND wi.wishlist_id = w.id AND w.workspace_id = $2
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: GetPriceHistoryByPlatform :many
SELECT wip.* FROM wishlist_item_prices wip
JOIN wishlist_items wi ON wi.id = wip.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE wip.item_id = $1 AND wip.platform_name = $2 AND w.workspace_id = $3
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY wip.price_date DESC, wip.created_at DESC;
