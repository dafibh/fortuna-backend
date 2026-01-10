-- +goose Up
-- +goose StatementBegin

-- Update recurring_transactions table for v2 Forward Financial Visibility
-- Add start_date and end_date for proper projection window management
-- Rename recurring_transaction_id to template_id for clarity

-- Add new date columns to recurring_transactions
ALTER TABLE recurring_transactions
ADD COLUMN start_date DATE,
ADD COLUMN end_date DATE;

-- Migrate due_day to start_date (use current date with that day-of-month)
-- For existing templates, we'll use the first occurrence after today with that due_day
UPDATE recurring_transactions
SET start_date = (
    CASE
        -- If due_day has already passed this month, start next month
        WHEN EXTRACT(DAY FROM CURRENT_DATE) > due_day THEN
            DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month' + (due_day - 1) * INTERVAL '1 day'
        -- Otherwise start this month
        ELSE
            DATE_TRUNC('month', CURRENT_DATE) + (due_day - 1) * INTERVAL '1 day'
    END
)::DATE
WHERE start_date IS NULL;

-- Now make start_date NOT NULL
ALTER TABLE recurring_transactions
ALTER COLUMN start_date SET NOT NULL;

-- end_date remains NULL (runs forever) for existing templates

-- Rename recurring_transaction_id to template_id in transactions table
ALTER TABLE transactions
RENAME COLUMN recurring_transaction_id TO template_id;

-- Update the index name to match
DROP INDEX IF EXISTS idx_transactions_recurring_id;
CREATE INDEX idx_transactions_template_id ON transactions(template_id) WHERE template_id IS NOT NULL;

COMMENT ON COLUMN recurring_transactions.start_date IS 'Date when recurring pattern starts. Day-of-month is used for monthly projections.';
COMMENT ON COLUMN recurring_transactions.end_date IS 'Optional end date. NULL means the recurring pattern runs forever.';
COMMENT ON COLUMN transactions.template_id IS 'Links transaction to recurring template if generated from recurring pattern.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Rename template_id back to recurring_transaction_id
DROP INDEX IF EXISTS idx_transactions_template_id;
ALTER TABLE transactions
RENAME COLUMN template_id TO recurring_transaction_id;
CREATE INDEX idx_transactions_recurring_id ON transactions(recurring_transaction_id) WHERE recurring_transaction_id IS NOT NULL;

-- Remove date columns from recurring_transactions
ALTER TABLE recurring_transactions
DROP COLUMN IF EXISTS end_date,
DROP COLUMN IF EXISTS start_date;

-- +goose StatementEnd
