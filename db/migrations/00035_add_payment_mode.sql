-- +goose Up
-- +goose StatementBegin
-- Add payment_mode column for consolidated loans feature
-- Determines how loan payments are tracked: per-item or consolidated monthly billing
ALTER TABLE loan_providers ADD COLUMN payment_mode TEXT NOT NULL DEFAULT 'per_item';

-- Add constraint to enforce valid values
ALTER TABLE loan_providers ADD CONSTRAINT payment_mode_check
    CHECK (payment_mode IN ('per_item', 'consolidated_monthly'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE loan_providers DROP CONSTRAINT IF EXISTS payment_mode_check;
ALTER TABLE loan_providers DROP COLUMN IF EXISTS payment_mode;
-- +goose StatementEnd
