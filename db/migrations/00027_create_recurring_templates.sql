-- +goose Up
-- Story 1.3: Create Recurring Templates Table
-- Creates table for recurring transaction patterns and adds FK constraint from transactions

CREATE TABLE recurring_templates (
    id SERIAL PRIMARY KEY,
    workspace_id INT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    category_id INT NOT NULL REFERENCES budget_categories(id),
    account_id INT NOT NULL REFERENCES accounts(id),
    frequency TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add CHECK constraint for valid frequency values
ALTER TABLE recurring_templates ADD CONSTRAINT chk_frequency
    CHECK (frequency IN ('monthly'));

-- Performance index for workspace queries
CREATE INDEX idx_recurring_templates_workspace_id ON recurring_templates(workspace_id);

-- Add FK constraint from transactions.template_id to recurring_templates.id
ALTER TABLE transactions
ADD CONSTRAINT fk_transactions_template_id
FOREIGN KEY (template_id) REFERENCES recurring_templates(id) ON DELETE SET NULL;

-- +goose Down
-- Must drop FK from transactions first, then drop table
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_template_id;
DROP INDEX IF EXISTS idx_recurring_templates_workspace_id;
DROP TABLE IF EXISTS recurring_templates;
