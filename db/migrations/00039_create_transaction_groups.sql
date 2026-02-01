-- +goose Up
-- +goose StatementBegin
-- Transaction Grouping: Create transaction_groups table
-- Groups are organizational containers for related transactions.
-- Group totals are ALWAYS derived via SUM(children) at query time.

CREATE TABLE transaction_groups (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
    name TEXT NOT NULL,
    month TEXT NOT NULL,
    auto_detected BOOLEAN NOT NULL DEFAULT false,
    loan_provider_id INTEGER NULL REFERENCES loan_providers(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Composite index for querying groups by workspace and month
CREATE INDEX idx_transaction_groups_workspace_month ON transaction_groups(workspace_id, month);

-- Index on workspace_id for general workspace queries
CREATE INDEX idx_transaction_groups_workspace_id ON transaction_groups(workspace_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_groups;
-- +goose StatementEnd
