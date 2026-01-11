-- +goose Up
-- Story 6.1: Add index for overdue CC detection query
-- Optimizes: SELECT * FROM transactions WHERE cc_state = 'billed' AND settlement_intent = 'deferred' AND billed_at < NOW() - 2 months
-- Partial index filters to only rows that could ever be overdue

CREATE INDEX IF NOT EXISTS idx_transactions_overdue
ON transactions(workspace_id, billed_at)
WHERE cc_state = 'billed' AND settlement_intent = 'deferred';

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_overdue;
