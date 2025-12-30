-- +goose Up
ALTER TABLE accounts ADD COLUMN deleted_at TIMESTAMPTZ NULL;
CREATE INDEX idx_accounts_deleted_at ON accounts(deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_accounts_deleted_at;
ALTER TABLE accounts DROP COLUMN IF EXISTS deleted_at;
