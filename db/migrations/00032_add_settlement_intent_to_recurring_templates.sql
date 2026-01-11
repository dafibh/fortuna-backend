-- +goose Up
-- +goose StatementBegin
-- Add settlement_intent column to recurring_templates for CC accounts
-- This allows users to specify default payment timing for projected CC transactions
ALTER TABLE recurring_templates
ADD COLUMN settlement_intent VARCHAR(20) DEFAULT NULL
CHECK (settlement_intent IS NULL OR settlement_intent IN ('immediate', 'deferred'));

COMMENT ON COLUMN recurring_templates.settlement_intent IS 'Default settlement intent for CC transactions: immediate (pay this month) or deferred (pay next month)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recurring_templates DROP COLUMN IF EXISTS settlement_intent;
-- +goose StatementEnd
