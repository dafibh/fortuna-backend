-- +goose Up
-- Drop deprecated v1 recurring and CC settlement objects
-- These have been replaced by v2:
--   - recurring_transactions -> recurring_templates
--   - transactions.recurring_transaction_id -> transactions.template_id
--   - transactions.cc_settlement_intent -> transactions.settlement_intent

-- Drop FK constraint and column from transactions
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_recurring_transaction_id_fkey;
DROP INDEX IF EXISTS idx_transactions_recurring_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS recurring_transaction_id;

-- Drop deprecated cc_settlement_intent column (replaced by settlement_intent in migration 00025)
ALTER TABLE transactions DROP COLUMN IF EXISTS cc_settlement_intent;

-- Drop the recurring_transactions table (replaced by recurring_templates in migration 00027)
DROP INDEX IF EXISTS idx_recurring_transactions_is_active;
DROP INDEX IF EXISTS idx_recurring_transactions_account_id;
DROP INDEX IF EXISTS idx_recurring_transactions_workspace_id;
DROP TABLE IF EXISTS recurring_transactions;

-- +goose Down
-- Recreate recurring_transactions table
CREATE TABLE recurring_transactions (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    category_id INTEGER REFERENCES budget_categories(id) ON DELETE SET NULL,
    frequency VARCHAR(20) NOT NULL DEFAULT 'monthly' CHECK (frequency = 'monthly'),
    due_day INTEGER NOT NULL DEFAULT 1 CHECK (due_day >= 1 AND due_day <= 31),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_recurring_transactions_workspace_id ON recurring_transactions(workspace_id);
CREATE INDEX idx_recurring_transactions_account_id ON recurring_transactions(account_id);
CREATE INDEX idx_recurring_transactions_is_active ON recurring_transactions(is_active) WHERE deleted_at IS NULL;

-- Recreate cc_settlement_intent column
ALTER TABLE transactions ADD COLUMN cc_settlement_intent VARCHAR(20) CHECK (cc_settlement_intent IN ('this_month', 'next_month'));

-- Recreate recurring_transaction_id column
ALTER TABLE transactions ADD COLUMN recurring_transaction_id INTEGER REFERENCES recurring_transactions(id) ON DELETE SET NULL;
CREATE INDEX idx_transactions_recurring_id ON transactions(recurring_transaction_id) WHERE recurring_transaction_id IS NOT NULL;
