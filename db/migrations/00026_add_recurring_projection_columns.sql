-- +goose Up
-- +goose StatementBegin

-- Add recurring/projection tracking columns for v2 Forward Financial Visibility
-- source: 'manual' (user created) or 'recurring' (generated from template)
-- is_projected: true = future projection, false = actual transaction

ALTER TABLE transactions
ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'recurring')),
ADD COLUMN is_projected BOOLEAN NOT NULL DEFAULT false;

-- Create index for projection queries
CREATE INDEX idx_transactions_is_projected ON transactions(is_projected) WHERE is_projected = true;
CREATE INDEX idx_transactions_source ON transactions(source) WHERE source = 'recurring';

-- Migrate existing data:
-- Transactions with recurring_transaction_id set are from recurring templates
UPDATE transactions
SET source = 'recurring'
WHERE recurring_transaction_id IS NOT NULL;

-- Note: All existing transactions are actual (not projected), so is_projected stays false

COMMENT ON COLUMN transactions.source IS 'Transaction source: manual (user created) or recurring (generated from template).';
COMMENT ON COLUMN transactions.is_projected IS 'True for future projected transactions, false for actual transactions.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_transactions_source;
DROP INDEX IF EXISTS idx_transactions_is_projected;

ALTER TABLE transactions
DROP COLUMN IF EXISTS is_projected,
DROP COLUMN IF EXISTS source;

-- +goose StatementEnd
