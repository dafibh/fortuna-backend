-- name: CreateLoanProvider :one
INSERT INTO loan_providers (
    workspace_id,
    name,
    cutoff_day,
    default_interest_rate
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetLoanProviderByID :one
SELECT * FROM loan_providers
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: ListLoanProviders :many
SELECT * FROM loan_providers
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY name ASC;

-- name: UpdateLoanProvider :one
UPDATE loan_providers
SET
    name = $3,
    cutoff_day = $4,
    default_interest_rate = $5,
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteLoanProvider :exec
UPDATE loan_providers
SET deleted_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- NOTE: CheckLoanProviderHasActiveLoans will be added in Story 7-2 when loans table exists
-- For now, delete without checking (no loans exist yet)
