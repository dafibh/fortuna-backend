-- name: GetWorkspaceByID :one
SELECT * FROM workspaces WHERE id = $1;

-- name: GetWorkspaceByUserID :one
SELECT * FROM workspaces WHERE user_id = $1;

-- name: GetWorkspaceByUserAuth0ID :one
SELECT w.* FROM workspaces w
INNER JOIN users u ON w.user_id = u.id
WHERE u.auth0_id = $1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (user_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE id = $1;
