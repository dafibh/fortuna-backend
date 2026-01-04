-- name: CreateWishlistItemNote :one
INSERT INTO wishlist_item_notes (item_id, content, image_path)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWishlistItemNoteByID :one
SELECT win.* FROM wishlist_item_notes win
JOIN wishlist_items wi ON wi.id = win.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE win.id = $1 AND w.workspace_id = $2 AND win.deleted_at IS NULL
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: ListNotesByItemAsc :many
SELECT win.* FROM wishlist_item_notes win
JOIN wishlist_items wi ON wi.id = win.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE win.item_id = $1 AND w.workspace_id = $2 AND win.deleted_at IS NULL
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY win.created_at ASC;

-- name: ListNotesByItemDesc :many
SELECT win.* FROM wishlist_item_notes win
JOIN wishlist_items wi ON wi.id = win.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE win.item_id = $1 AND w.workspace_id = $2 AND win.deleted_at IS NULL
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY win.created_at DESC;

-- name: CountNotesByItem :one
SELECT COUNT(*) FROM wishlist_item_notes win
JOIN wishlist_items wi ON wi.id = win.item_id
JOIN wishlists w ON w.id = wi.wishlist_id
WHERE win.item_id = $1 AND w.workspace_id = $2 AND win.deleted_at IS NULL
AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: UpdateWishlistItemNote :one
UPDATE wishlist_item_notes win
SET content = $3, image_path = $4, updated_at = NOW()
FROM wishlist_items wi, wishlists w
WHERE win.id = $1 AND win.item_id = wi.id AND wi.wishlist_id = w.id AND w.workspace_id = $2
AND win.deleted_at IS NULL AND wi.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING win.*;

-- name: DeleteWishlistItemNote :exec
UPDATE wishlist_item_notes win
SET deleted_at = NOW()
FROM wishlist_items wi, wishlists w
WHERE win.id = $1 AND win.item_id = wi.id AND wi.wishlist_id = w.id AND w.workspace_id = $2
AND win.deleted_at IS NULL AND wi.deleted_at IS NULL AND w.deleted_at IS NULL;
