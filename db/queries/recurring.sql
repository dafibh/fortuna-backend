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

-- ========================================
-- V2 Recurring Templates (recurring_templates table)
-- ========================================

-- name: CreateRecurringTemplate :one
INSERT INTO recurring_templates (
    workspace_id, description, amount, category_id, account_id,
    frequency, start_date, end_date
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateRecurringTemplate :one
UPDATE recurring_templates
SET description = $3, amount = $4, category_id = $5, account_id = $6,
    frequency = $7, start_date = $8, end_date = $9, updated_at = NOW()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteRecurringTemplate :exec
DELETE FROM recurring_templates
WHERE id = $1 AND workspace_id = $2;

-- name: GetRecurringTemplateByID :one
SELECT * FROM recurring_templates
WHERE id = $1 AND workspace_id = $2;

-- name: ListRecurringTemplatesByWorkspace :many
SELECT * FROM recurring_templates
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: GetActiveRecurringTemplates :many
SELECT * FROM recurring_templates
WHERE workspace_id = $1
  AND (end_date IS NULL OR end_date >= CURRENT_DATE)
ORDER BY start_date;
