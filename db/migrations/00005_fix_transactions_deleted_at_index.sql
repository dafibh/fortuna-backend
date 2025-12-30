-- +goose Up
-- Fix partial index on deleted_at to help queries finding ACTIVE records
-- The previous index helped find DELETED records, but all queries filter for active (deleted_at IS NULL)

DROP INDEX IF EXISTS idx_transactions_deleted_at;
CREATE INDEX idx_transactions_active ON transactions(workspace_id, transaction_date) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_active;
CREATE INDEX idx_transactions_deleted_at ON transactions(deleted_at) WHERE deleted_at IS NOT NULL;
