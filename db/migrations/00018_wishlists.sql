-- +goose Up
-- +goose StatementBegin
CREATE TABLE wishlists (
    id SERIAL PRIMARY KEY,
    workspace_id INT NOT NULL REFERENCES workspaces(id),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Unique constraint on (workspace_id, name) for non-deleted records
CREATE UNIQUE INDEX idx_wishlists_workspace_name
    ON wishlists(workspace_id, name)
    WHERE deleted_at IS NULL;

-- Index for workspace filtering
CREATE INDEX idx_wishlists_workspace
    ON wishlists(workspace_id)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wishlists_workspace;
DROP INDEX IF EXISTS idx_wishlists_workspace_name;
DROP TABLE IF EXISTS wishlists;
-- +goose StatementEnd
