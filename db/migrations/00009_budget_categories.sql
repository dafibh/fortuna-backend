-- +goose Up
-- +goose StatementBegin
CREATE TABLE budget_categories (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

-- Unique name per workspace for active (non-deleted) categories
-- NULLS NOT DISTINCT ensures (workspace_id, name, NULL) is treated as unique
CREATE UNIQUE INDEX idx_budget_categories_unique_name
ON budget_categories(workspace_id, name)
WHERE deleted_at IS NULL;

-- Index for efficient workspace queries
CREATE INDEX idx_budget_categories_workspace
ON budget_categories(workspace_id)
WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_budget_categories_workspace;
DROP INDEX IF EXISTS idx_budget_categories_unique_name;
DROP TABLE IF EXISTS budget_categories;
-- +goose StatementEnd
