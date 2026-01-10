-- +goose Up
-- +goose StatementBegin

-- Add unique constraint to prevent duplicate projections for same template + month
-- This prevents race conditions when concurrent projection generation occurs
-- Uses partial index: only applies to is_projected=true transactions with a template_id
-- Note: Using EXTRACT (year/month) instead of DATE_TRUNC because EXTRACT is immutable
CREATE UNIQUE INDEX idx_transactions_projection_unique
ON transactions(workspace_id, template_id, EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date))
WHERE is_projected = true AND template_id IS NOT NULL AND deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_projection_unique;
-- +goose StatementEnd
