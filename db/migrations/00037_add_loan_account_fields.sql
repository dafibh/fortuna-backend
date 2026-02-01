-- +goose Up
-- +goose StatementBegin
-- Add account_id and settlement_intent to loans table for consolidated loans v2
-- These fields determine where loan payments are recorded and how CC payments are timed

-- Add account_id column (nullable initially for existing loans)
ALTER TABLE loans ADD COLUMN account_id INTEGER;

-- Add settlement_intent column (null for bank accounts, 'immediate' or 'deferred' for CC)
ALTER TABLE loans ADD COLUMN settlement_intent TEXT;

-- Add CHECK constraint for settlement_intent valid values
ALTER TABLE loans ADD CONSTRAINT chk_loans_settlement_intent
    CHECK (settlement_intent IS NULL OR settlement_intent IN ('immediate', 'deferred'));

-- Add foreign key to accounts (RESTRICT prevents deleting accounts with active loans)
ALTER TABLE loans ADD CONSTRAINT fk_loans_account
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT;

-- Create index for account lookups
CREATE INDEX idx_loans_account ON loans(account_id) WHERE deleted_at IS NULL;

-- NOTE: After backfilling existing loans with account_id, you should add NOT NULL constraint:
-- ALTER TABLE loans ALTER COLUMN account_id SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_loans_account;
ALTER TABLE loans DROP CONSTRAINT IF EXISTS fk_loans_account;
ALTER TABLE loans DROP CONSTRAINT IF EXISTS chk_loans_settlement_intent;
ALTER TABLE loans DROP COLUMN IF EXISTS settlement_intent;
ALTER TABLE loans DROP COLUMN IF EXISTS account_id;
-- +goose StatementEnd
