-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
ADD COLUMN category_id INTEGER NULL REFERENCES budget_categories(id);

-- Index for query performance (partial index for non-null values)
CREATE INDEX idx_transactions_category
ON transactions(category_id)
WHERE category_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_category;
ALTER TABLE transactions DROP COLUMN IF EXISTS category_id;
-- +goose StatementEnd
