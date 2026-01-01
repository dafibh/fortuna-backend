-- name: UpsertBudgetAllocation :one
INSERT INTO budget_allocations (workspace_id, category_id, year, month, amount)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (workspace_id, category_id, year, month)
DO UPDATE SET amount = EXCLUDED.amount, updated_at = NOW()
RETURNING *;

-- name: GetBudgetAllocationsByMonth :many
SELECT ba.*, bc.name AS category_name
FROM budget_allocations ba
JOIN budget_categories bc ON ba.category_id = bc.id
WHERE ba.workspace_id = $1 AND ba.year = $2 AND ba.month = $3
ORDER BY bc.name ASC;

-- name: GetBudgetAllocationByCategory :one
SELECT * FROM budget_allocations
WHERE workspace_id = $1 AND category_id = $2 AND year = $3 AND month = $4;

-- name: DeleteBudgetAllocation :exec
DELETE FROM budget_allocations
WHERE workspace_id = $1 AND category_id = $2 AND year = $3 AND month = $4;

-- name: GetCategoriesWithAllocations :many
-- Returns all categories with their allocation for a specific month (0 if not set)
SELECT
    bc.id AS category_id,
    bc.name AS category_name,
    COALESCE(ba.amount, 0) AS allocated
FROM budget_categories bc
LEFT JOIN budget_allocations ba ON bc.id = ba.category_id
    AND ba.year = $2 AND ba.month = $3 AND ba.workspace_id = $1
WHERE bc.workspace_id = $1 AND bc.deleted_at IS NULL
ORDER BY bc.name ASC;
