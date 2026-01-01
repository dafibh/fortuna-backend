-- name: CreateBudgetCategory :one
INSERT INTO budget_categories (workspace_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetBudgetCategoryByID :one
SELECT * FROM budget_categories
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: GetAllBudgetCategories :many
SELECT * FROM budget_categories
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY name ASC;

-- name: UpdateBudgetCategory :one
UPDATE budget_categories
SET name = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteBudgetCategory :exec
UPDATE budget_categories
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: CountTransactionsByCategory :one
-- NOTE: This query will be valid after Story 4.2 adds category_id to transactions
-- For now, return 0 (no transactions can have categories yet)
SELECT 0::bigint AS count;

-- name: GetBudgetCategoryByName :one
SELECT * FROM budget_categories
WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL;
