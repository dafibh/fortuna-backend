-- name: CreateRecurringTransaction :one
INSERT INTO recurring_transactions (
    workspace_id, name, amount, account_id, type, category_id, frequency, due_day, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetRecurringTransaction :one
SELECT * FROM recurring_transactions
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: ListRecurringTransactions :many
SELECT * FROM recurring_transactions
WHERE workspace_id = $1 AND deleted_at IS NULL
    AND ($2::BOOLEAN IS NULL OR is_active = $2)
ORDER BY name ASC;

-- name: UpdateRecurringTransaction :one
UPDATE recurring_transactions
SET name = $3, amount = $4, account_id = $5, type = $6, category_id = $7,
    frequency = $8, due_day = $9, is_active = $10, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteRecurringTransaction :execrows
UPDATE recurring_transactions
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: CheckRecurringTransactionExists :one
-- Check if a transaction already exists for a recurring template in a specific month
SELECT COUNT(*)::INTEGER as count
FROM transactions
WHERE recurring_transaction_id = sqlc.arg(recurring_id)::INTEGER
    AND workspace_id = sqlc.arg(workspace_id)::INTEGER
    AND EXTRACT(YEAR FROM transaction_date) = sqlc.arg(year)::INTEGER
    AND EXTRACT(MONTH FROM transaction_date) = sqlc.arg(month)::INTEGER
    AND deleted_at IS NULL;
