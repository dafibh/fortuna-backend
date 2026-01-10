-- +goose Up
-- Story 1.2: Add Recurring/Projection Columns to Transactions Table
-- Adds columns to identify transaction source and distinguish projections from actuals

ALTER TABLE transactions ADD COLUMN source TEXT DEFAULT 'manual';
ALTER TABLE transactions ADD COLUMN template_id INT NULL;
ALTER TABLE transactions ADD COLUMN is_projected BOOLEAN DEFAULT false;

-- Add CHECK constraint for valid source values
ALTER TABLE transactions ADD CONSTRAINT chk_source
    CHECK (source IN ('manual', 'recurring'));

-- Index for querying by source and projected status
CREATE INDEX idx_transactions_source ON transactions(source);
CREATE INDEX idx_transactions_template_id ON transactions(template_id) WHERE template_id IS NOT NULL;
CREATE INDEX idx_transactions_is_projected ON transactions(is_projected) WHERE is_projected = true;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_is_projected;
DROP INDEX IF EXISTS idx_transactions_template_id;
DROP INDEX IF EXISTS idx_transactions_source;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_source;
ALTER TABLE transactions DROP COLUMN IF EXISTS is_projected;
ALTER TABLE transactions DROP COLUMN IF EXISTS template_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS source;
