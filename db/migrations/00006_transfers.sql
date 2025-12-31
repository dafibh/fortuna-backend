-- +goose Up
ALTER TABLE transactions ADD COLUMN transfer_pair_id UUID NULL;
CREATE INDEX idx_transactions_transfer_pair_id ON transactions(transfer_pair_id) WHERE transfer_pair_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_transfer_pair_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS transfer_pair_id;
