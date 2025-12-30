-- name: CreateAccount :one
INSERT INTO accounts (workspace_id, name, account_type, template, initial_balance)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE workspace_id = $1 AND id = $2;

-- name: GetAccountsByWorkspace :many
SELECT * FROM accounts WHERE workspace_id = $1 ORDER BY created_at DESC;
