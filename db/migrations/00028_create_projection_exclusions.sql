-- +goose Up
-- +goose StatementBegin

-- Tracks explicitly deleted projected transactions to prevent re-creation on sync
CREATE TABLE projection_exclusions (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    template_id INTEGER NOT NULL REFERENCES recurring_templates(id) ON DELETE CASCADE,
    excluded_month DATE NOT NULL,  -- First day of excluded month
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workspace_id, template_id, excluded_month)
);

-- Index for efficient exclusion checks during projection generation
CREATE INDEX idx_projection_exclusions_template ON projection_exclusions(template_id, excluded_month);
CREATE INDEX idx_projection_exclusions_workspace ON projection_exclusions(workspace_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_projection_exclusions_workspace;
DROP INDEX IF EXISTS idx_projection_exclusions_template;
DROP TABLE IF EXISTS projection_exclusions;
-- +goose StatementEnd
