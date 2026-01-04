-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, workspace_id, description, token_hash, token_prefix)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAPITokensByWorkspace :many
SELECT * FROM api_tokens
WHERE workspace_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetAPITokenByID :one
SELECT * FROM api_tokens
WHERE workspace_id = $1 AND id = $2;

-- name: GetAPITokenByHash :one
SELECT * FROM api_tokens
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: RevokeAPIToken :execrows
UPDATE api_tokens
SET revoked_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND revoked_at IS NULL;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = NOW()
WHERE id = $1;
