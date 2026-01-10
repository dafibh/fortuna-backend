-- name: CreateRecurringTransaction :one
INSERT INTO recurring_transactions (
    workspace_id, name, amount, account_id, type, category_id, frequency, due_day, is_active, start_date, end_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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
    frequency = $8, due_day = $9, is_active = $10, start_date = $11, end_date = $12, updated_at = NOW()
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
WHERE template_id = sqlc.arg(recurring_id)::INTEGER
    AND workspace_id = sqlc.arg(workspace_id)::INTEGER
    AND EXTRACT(YEAR FROM transaction_date) = sqlc.arg(year)::INTEGER
    AND EXTRACT(MONTH FROM transaction_date) = sqlc.arg(month)::INTEGER
    AND deleted_at IS NULL;

-- =====================================================
-- V2 RECURRING TEMPLATE QUERIES
-- =====================================================

-- name: GetActiveTemplates :many
-- Get all active recurring templates (no end_date or end_date in future)
SELECT * FROM recurring_transactions
WHERE workspace_id = $1
    AND deleted_at IS NULL
    AND is_active = true
    AND (end_date IS NULL OR end_date >= CURRENT_DATE)
ORDER BY name ASC;

-- name: GetTemplatesByWorkspace :many
-- Get all recurring templates for a workspace (including inactive)
SELECT * FROM recurring_transactions
WHERE workspace_id = $1
    AND deleted_at IS NULL
ORDER BY is_active DESC, name ASC;

-- name: GetTemplateWithProjectionRange :one
-- Get template with the date range of its projections
SELECT rt.*,
    MIN(t.transaction_date) as first_projection_date,
    MAX(t.transaction_date) as last_projection_date
FROM recurring_transactions rt
LEFT JOIN transactions t ON t.template_id = rt.id
    AND t.is_projected = true
    AND t.deleted_at IS NULL
WHERE rt.workspace_id = $1 AND rt.id = $2 AND rt.deleted_at IS NULL
GROUP BY rt.id;

-- name: SetTemplateEndDate :one
-- Set or update the end_date for a recurring template
UPDATE recurring_transactions
SET end_date = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: ToggleTemplateActive :one
-- Toggle the is_active status of a template
UPDATE recurring_transactions
SET is_active = NOT is_active, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;
