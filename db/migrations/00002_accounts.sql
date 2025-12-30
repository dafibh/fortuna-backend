-- +goose Up
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(20) NOT NULL CHECK (account_type IN ('asset', 'liability')),
    template VARCHAR(20) NOT NULL CHECK (template IN ('bank', 'cash', 'ewallet', 'credit_card')),
    initial_balance NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_workspace_id ON accounts(workspace_id);

-- +goose Down
DROP TABLE IF EXISTS accounts;
