-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
ADD COLUMN recurring_transaction_id INTEGER REFERENCES recurring_transactions(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_recurring_id ON transactions(recurring_transaction_id) WHERE recurring_transaction_id IS NOT NULL;

COMMENT ON COLUMN transactions.recurring_transaction_id IS 'Links transaction to recurring template if auto-generated';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_recurring_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS recurring_transaction_id;
-- +goose StatementEnd
