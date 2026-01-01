-- name: CreateAccount :one
INSERT INTO accounts (workspace_id, name, account_type, template, initial_balance)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: GetAccountByIDIncludeDeleted :one
SELECT * FROM accounts
WHERE workspace_id = $1 AND id = $2;

-- name: GetAccountsByWorkspace :many
SELECT * FROM accounts
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAccountsByWorkspaceAll :many
SELECT * FROM accounts
WHERE workspace_id = $1
ORDER BY deleted_at NULLS FIRST, created_at DESC;

-- name: UpdateAccount :one
UPDATE accounts
SET name = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAccount :execrows
UPDATE accounts
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: HardDeleteAccount :exec
DELETE FROM accounts
WHERE workspace_id = $1 AND id = $2;

-- name: GetCCOutstandingSummary :one
-- Get total outstanding balance across all CC accounts (sum of unpaid expenses)
SELECT
    COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total_outstanding,
    COUNT(DISTINCT a.id)::INTEGER as cc_account_count
FROM accounts a
LEFT JOIN transactions t ON t.account_id = a.id
    AND t.type = 'expense'
    AND t.is_paid = false
    AND t.deleted_at IS NULL
WHERE a.workspace_id = $1
    AND a.template = 'credit_card'
    AND a.deleted_at IS NULL;

-- name: GetPerAccountOutstanding :many
-- Get outstanding balance for each CC account
SELECT
    a.id,
    a.name,
    COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as outstanding_balance
FROM accounts a
LEFT JOIN transactions t ON t.account_id = a.id
    AND t.type = 'expense'
    AND t.is_paid = false
    AND t.deleted_at IS NULL
WHERE a.workspace_id = $1
    AND a.template = 'credit_card'
    AND a.deleted_at IS NULL
GROUP BY a.id, a.name
ORDER BY a.name;
