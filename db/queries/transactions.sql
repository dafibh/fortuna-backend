-- name: CreateTransaction :one
INSERT INTO transactions (
    workspace_id, account_id, name, amount, type,
    transaction_date, is_paid, cc_settlement_intent, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: GetTransactionsByWorkspace :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND deleted_at IS NULL
  AND ($2::INTEGER IS NULL OR account_id = $2)
  AND ($3::DATE IS NULL OR transaction_date >= $3)
  AND ($4::DATE IS NULL OR transaction_date <= $4)
  AND ($5::VARCHAR IS NULL OR type = $5)
ORDER BY transaction_date DESC, created_at DESC
LIMIT $6 OFFSET $7;

-- name: CountTransactionsByWorkspace :one
SELECT COUNT(*) FROM transactions
WHERE workspace_id = $1
  AND deleted_at IS NULL
  AND ($2::INTEGER IS NULL OR account_id = $2)
  AND ($3::DATE IS NULL OR transaction_date >= $3)
  AND ($4::DATE IS NULL OR transaction_date <= $4)
  AND ($5::VARCHAR IS NULL OR type = $5);
