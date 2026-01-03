-- name: CreateWishlist :one
INSERT INTO wishlists (workspace_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetWishlistByID :one
SELECT * FROM wishlists
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: ListWishlists :many
SELECT * FROM wishlists
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateWishlist :one
UPDATE wishlists
SET name = $3, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWishlist :exec
UPDATE wishlists
SET deleted_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetWishlistByName :one
SELECT * FROM wishlists
WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL;

-- NOTE: ListWishlistsWithItemCount and CountWishlistItems deferred to Story 8-2
-- when wishlist_items table is created
