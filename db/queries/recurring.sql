-- Recurring Templates (recurring_templates table)

-- name: CreateRecurringTemplate :one
INSERT INTO recurring_templates (
    workspace_id, description, amount, category_id, account_id,
    frequency, start_date, end_date, notes, settlement_intent
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateRecurringTemplate :one
UPDATE recurring_templates
SET description = $3, amount = $4, category_id = $5, account_id = $6,
    frequency = $7, start_date = $8, end_date = $9, notes = $10, settlement_intent = $11, updated_at = NOW()
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

-- name: GetAllActiveTemplates :many
-- Get all active templates across all workspaces (for daily sync goroutine)
SELECT * FROM recurring_templates
WHERE end_date IS NULL OR end_date >= CURRENT_DATE
ORDER BY workspace_id, id;
