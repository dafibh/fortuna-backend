-- +goose Up
-- Story 1.1: Add CC Lifecycle Columns to Transactions Table
-- Adds columns to track pending, billed, and settled states for credit card transactions

ALTER TABLE transactions ADD COLUMN cc_state TEXT;
ALTER TABLE transactions ADD COLUMN billed_at TIMESTAMPTZ NULL;
ALTER TABLE transactions ADD COLUMN settled_at TIMESTAMPTZ NULL;
ALTER TABLE transactions ADD COLUMN settlement_intent TEXT;

-- Add CHECK constraints for valid values
ALTER TABLE transactions ADD CONSTRAINT chk_cc_state
    CHECK (cc_state IS NULL OR cc_state IN ('pending', 'billed', 'settled'));
ALTER TABLE transactions ADD CONSTRAINT chk_settlement_intent
    CHECK (settlement_intent IS NULL OR settlement_intent IN ('immediate', 'deferred'));

-- Index for querying CC transactions by state
CREATE INDEX idx_transactions_cc_state ON transactions(cc_state) WHERE cc_state IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_cc_state;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_settlement_intent;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_cc_state;
ALTER TABLE transactions DROP COLUMN IF EXISTS settlement_intent;
ALTER TABLE transactions DROP COLUMN IF EXISTS settled_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS billed_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS cc_state;
