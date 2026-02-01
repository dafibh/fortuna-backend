-- +goose Up
-- +goose StatementBegin
-- Add 'loan' as a valid source type for transactions
-- This supports the Consolidated Loans v2 feature where loan payments are stored as transactions

-- Drop the existing constraint
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_source;

-- Recreate with 'loan' included
ALTER TABLE transactions ADD CONSTRAINT chk_source
    CHECK (source IN ('manual', 'recurring', 'loan'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to original constraint (will fail if any 'loan' source transactions exist)
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_source;

ALTER TABLE transactions ADD CONSTRAINT chk_source
    CHECK (source IN ('manual', 'recurring'));
-- +goose StatementEnd
