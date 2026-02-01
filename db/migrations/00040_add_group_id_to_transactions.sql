-- +goose Up
-- +goose StatementBegin
-- Transaction Grouping: Add group_id FK to transactions table
-- Existing transactions remain unaffected (group_id defaults to NULL).

-- Step 1: Add nullable group_id column
ALTER TABLE transactions ADD COLUMN group_id INTEGER NULL;

-- Step 2: Add FK constraint with ON DELETE SET NULL
-- When a group is deleted, child transactions become ungrouped automatically
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_group
    FOREIGN KEY (group_id) REFERENCES transaction_groups(id) ON DELETE SET NULL;

-- Step 3: Partial index on non-null group_id values only
-- Most transactions are ungrouped, so only index those that belong to a group
CREATE INDEX idx_transactions_group_id ON transactions(group_id)
    WHERE group_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_group_id;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_group;
ALTER TABLE transactions DROP COLUMN IF EXISTS group_id;
-- +goose StatementEnd
