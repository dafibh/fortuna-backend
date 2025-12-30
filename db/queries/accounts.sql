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

-- name: SoftDeleteAccount :exec
UPDATE accounts
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: HardDeleteAccount :exec
DELETE FROM accounts
WHERE workspace_id = $1 AND id = $2;
