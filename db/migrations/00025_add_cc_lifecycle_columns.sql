-- +goose Up

-- Add CC lifecycle columns for v2 Forward Financial Visibility
-- cc_state: 'pending' (purchased, not yet billed), 'billed' (in outstanding), 'settled' (paid off)
-- settlement_intent: 'immediate' (pay now) or 'deferred' (pay later)

-- +goose StatementBegin
ALTER TABLE transactions
ADD COLUMN cc_state VARCHAR(20) CHECK (cc_state IN ('pending', 'billed', 'settled')),
ADD COLUMN billed_at TIMESTAMPTZ,
ADD COLUMN settled_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_transactions_cc_state ON transactions(cc_state) WHERE cc_state IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_transactions_billed_at ON transactions(billed_at) WHERE billed_at IS NOT NULL;
-- +goose StatementEnd

-- Migrate existing CC transaction data
-- First, update CC transactions that are paid (settled)
-- +goose StatementBegin
UPDATE transactions t
SET
    cc_state = 'settled',
    settled_at = t.updated_at
FROM accounts a
WHERE t.account_id = a.id
    AND a.template = 'credit_card'
    AND a.deleted_at IS NULL
    AND t.type = 'expense'
    AND t.is_paid = true
    AND t.deleted_at IS NULL;
-- +goose StatementEnd

-- Update CC transactions that are unpaid (billed - since they existed in v1, they're already "billed")
-- +goose StatementBegin
UPDATE transactions t
SET
    cc_state = 'billed',
    billed_at = t.created_at
FROM accounts a
WHERE t.account_id = a.id
    AND a.template = 'credit_card'
    AND a.deleted_at IS NULL
    AND t.type = 'expense'
    AND t.is_paid = false
    AND t.deleted_at IS NULL;
-- +goose StatementEnd

-- DROP the old constraint FIRST before updating values
-- +goose StatementBegin
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_cc_settlement_intent_check;
-- +goose StatementEnd

-- Now migrate cc_settlement_intent values: this_month -> immediate, next_month -> deferred
-- +goose StatementBegin
UPDATE transactions
SET cc_settlement_intent = 'immediate'
WHERE cc_settlement_intent = 'this_month';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE transactions
SET cc_settlement_intent = 'deferred'
WHERE cc_settlement_intent = 'next_month';
-- +goose StatementEnd

-- Add the new constraint with updated values
-- +goose StatementBegin
ALTER TABLE transactions
ADD CONSTRAINT transactions_cc_settlement_intent_check
CHECK (cc_settlement_intent IN ('immediate', 'deferred'));
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON COLUMN transactions.cc_state IS 'CC lifecycle state: pending (not yet billed), billed (in outstanding), settled (paid off). NULL for non-CC transactions.';
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON COLUMN transactions.billed_at IS 'Timestamp when CC transaction was marked as billed (appeared in banking app outstanding).';
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON COLUMN transactions.settled_at IS 'Timestamp when CC transaction was settled via payment.';
-- +goose StatementEnd

-- +goose StatementBegin
COMMENT ON COLUMN transactions.cc_settlement_intent IS 'Settlement intent: immediate (pay now) or deferred (pay later). NULL for non-CC transactions.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Revert settlement_intent values
UPDATE transactions
SET cc_settlement_intent = 'this_month'
WHERE cc_settlement_intent = 'immediate';

UPDATE transactions
SET cc_settlement_intent = 'next_month'
WHERE cc_settlement_intent = 'deferred';

-- Restore original check constraint
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS transactions_cc_settlement_intent_check;

ALTER TABLE transactions
ADD CONSTRAINT transactions_cc_settlement_intent_check
CHECK (cc_settlement_intent IN ('this_month', 'next_month'));

-- Drop new columns
DROP INDEX IF EXISTS idx_transactions_billed_at;
DROP INDEX IF EXISTS idx_transactions_cc_state;

ALTER TABLE transactions
DROP COLUMN IF EXISTS settled_at,
DROP COLUMN IF EXISTS billed_at,
DROP COLUMN IF EXISTS cc_state;

-- +goose StatementEnd
