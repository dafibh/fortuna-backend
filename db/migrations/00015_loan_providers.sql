-- +goose Up
-- +goose StatementBegin
CREATE TABLE loan_providers (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
    name VARCHAR(100) NOT NULL,
    cutoff_day INTEGER NOT NULL CHECK (cutoff_day >= 1 AND cutoff_day <= 31),
    default_interest_rate NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_loan_providers_unique_name ON loan_providers(workspace_id, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_loan_providers_workspace ON loan_providers(workspace_id) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_loan_providers_workspace;
DROP INDEX IF EXISTS idx_loan_providers_unique_name;
DROP TABLE IF EXISTS loan_providers;
-- +goose StatementEnd
