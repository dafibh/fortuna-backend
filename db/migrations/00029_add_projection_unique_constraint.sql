-- +goose Up
-- +goose StatementBegin

-- Add unique constraint to prevent duplicate projections for same template + month
-- This prevents race conditions when concurrent projection generation occurs
-- Uses partial index: only applies to is_projected=true transactions with a template_id
CREATE UNIQUE INDEX idx_transactions_projection_unique
ON transactions(workspace_id, template_id, DATE_TRUNC('month', transaction_date))
WHERE is_projected = true AND template_id IS NOT NULL AND deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_projection_unique;
-- +goose StatementEnd
