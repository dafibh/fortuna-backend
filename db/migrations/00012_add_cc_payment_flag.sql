-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN is_cc_payment BOOLEAN NOT NULL DEFAULT false;

-- Add partial index for CC payment queries
CREATE INDEX idx_transactions_cc_payment ON transactions(account_id, is_cc_payment) WHERE is_cc_payment = true AND deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_cc_payment;
ALTER TABLE transactions DROP COLUMN is_cc_payment;
-- +goose StatementEnd
