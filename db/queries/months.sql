-- name: CreateMonth :one
INSERT INTO months (workspace_id, year, month, start_date, end_date, starting_balance)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetMonthByYearMonth :one
SELECT * FROM months
WHERE workspace_id = $1 AND year = $2 AND month = $3;

-- name: GetLatestMonth :one
SELECT * FROM months
WHERE workspace_id = $1
ORDER BY year DESC, month DESC
LIMIT 1;

-- name: GetAllMonths :many
SELECT * FROM months
WHERE workspace_id = $1
ORDER BY year DESC, month DESC;

-- name: UpdateMonthStartingBalance :exec
UPDATE months
SET starting_balance = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2;
