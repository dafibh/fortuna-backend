-- name: CreateGroup :one
INSERT INTO transaction_groups (
    workspace_id, name, month, auto_detected, loan_provider_id
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetGroupByID :one
SELECT tg.id, tg.workspace_id, tg.name, tg.month,
       tg.auto_detected, tg.loan_provider_id,
       tg.created_at, tg.updated_at,
       COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total_amount,
       COUNT(t.id)::INTEGER as child_count
FROM transaction_groups tg
LEFT JOIN transactions t ON t.group_id = tg.id AND t.deleted_at IS NULL
WHERE tg.workspace_id = $1 AND tg.id = $2
GROUP BY tg.id;

-- name: GetGroupsByMonth :many
SELECT tg.id, tg.workspace_id, tg.name, tg.month,
       tg.auto_detected, tg.loan_provider_id,
       tg.created_at, tg.updated_at,
       COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total_amount,
       COUNT(t.id)::INTEGER as child_count
FROM transaction_groups tg
LEFT JOIN transactions t ON t.group_id = tg.id AND t.deleted_at IS NULL
WHERE tg.workspace_id = $1 AND tg.month = $2
GROUP BY tg.id
ORDER BY tg.created_at DESC;

-- name: UpdateGroupName :one
UPDATE transaction_groups
SET name = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2
RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM transaction_groups
WHERE workspace_id = $1 AND id = $2;

-- name: AssignGroupToTransactions :exec
UPDATE transactions
SET group_id = $1, updated_at = NOW()
WHERE workspace_id = $2
  AND id = ANY($3::int[])
  AND deleted_at IS NULL;

-- name: UnassignGroupFromTransactions :exec
UPDATE transactions
SET group_id = NULL, updated_at = NOW()
WHERE workspace_id = $1
  AND id = ANY($2::int[])
  AND deleted_at IS NULL;

-- name: GetUngroupedTransactionsByMonth :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND group_id IS NULL
  AND transaction_date >= $2
  AND transaction_date < $3
  AND deleted_at IS NULL
ORDER BY transaction_date DESC, created_at DESC;

-- name: UnassignAllFromGroup :execrows
UPDATE transactions SET group_id = NULL, updated_at = NOW()
WHERE group_id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: SoftDeleteTransactionsByGroupID :execrows
UPDATE transactions SET deleted_at = NOW(), updated_at = NOW()
WHERE group_id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: CountGroupChildren :one
SELECT COUNT(*)::INTEGER as child_count
FROM transactions
WHERE group_id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: GetAutoDetectedGroupByProviderMonth :one
SELECT tg.id, tg.workspace_id, tg.name, tg.month,
       tg.auto_detected, tg.loan_provider_id,
       tg.created_at, tg.updated_at,
       COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total_amount,
       COUNT(t.id)::INTEGER as child_count
FROM transaction_groups tg
LEFT JOIN transactions t ON t.group_id = tg.id AND t.deleted_at IS NULL
WHERE tg.workspace_id = $1
  AND tg.auto_detected = true
  AND tg.loan_provider_id = $2
  AND tg.month = $3
GROUP BY tg.id;

-- name: GetConsolidatedProvidersByMonth :many
SELECT lp.id as provider_id, lp.name as provider_name, COUNT(t.id)::INTEGER as tx_count
FROM transactions t
JOIN loans l ON t.loan_id = l.id
JOIN loan_providers lp ON l.provider_id = lp.id
WHERE t.workspace_id = @workspace_id
  AND lp.payment_mode = 'consolidated_monthly'
  AND t.group_id IS NULL
  AND t.deleted_at IS NULL
  AND TO_CHAR(t.transaction_date, 'YYYY-MM') = @month::TEXT
GROUP BY lp.id, lp.name
HAVING COUNT(t.id) >= 2;

-- name: GetUngroupedTransactionIDsByProviderMonth :many
SELECT t.id
FROM transactions t
JOIN loans l ON t.loan_id = l.id
WHERE t.workspace_id = @workspace_id
  AND l.provider_id = @provider_id
  AND t.group_id IS NULL
  AND t.deleted_at IS NULL
  AND TO_CHAR(t.transaction_date, 'YYYY-MM') = @month::TEXT;
