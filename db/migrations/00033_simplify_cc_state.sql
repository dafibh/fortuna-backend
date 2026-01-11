-- +goose Up
-- +goose StatementBegin
-- Simplify CC transaction model: use isPaid for settlement instead of ccState

-- First, ensure isPaid is true for all settled CC transactions
UPDATE transactions
SET is_paid = true
WHERE cc_state = 'settled' AND is_paid = false;

-- Drop the cc_state column (no longer needed - isPaid indicates settlement)
ALTER TABLE transactions DROP COLUMN IF EXISTS cc_state;

-- Drop the settled_at column (no longer needed - use updated_at or is_paid timestamp if needed)
ALTER TABLE transactions DROP COLUMN IF EXISTS settled_at;

-- Add comment explaining the simplified model
COMMENT ON COLUMN transactions.is_paid IS 'For CC transactions: true = settled (bill paid). For regular transactions: true = paid/completed.';
COMMENT ON COLUMN transactions.billed_at IS 'For CC transactions only: timestamp when transaction appeared on statement. NULL = pending, set = billed.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Re-add the columns (data will be lost)
ALTER TABLE transactions ADD COLUMN cc_state VARCHAR(20) DEFAULT NULL;
ALTER TABLE transactions ADD COLUMN settled_at TIMESTAMPTZ DEFAULT NULL;

-- Add back the check constraint
ALTER TABLE transactions ADD CONSTRAINT transactions_cc_state_check
CHECK (cc_state IS NULL OR cc_state IN ('pending', 'billed', 'settled'));

-- Attempt to restore cc_state based on current data
UPDATE transactions
SET cc_state = CASE
    WHEN is_paid = true AND billed_at IS NOT NULL THEN 'settled'
    WHEN billed_at IS NOT NULL THEN 'billed'
    ELSE 'pending'
END
WHERE billed_at IS NOT NULL OR is_paid = true;
-- +goose StatementEnd
