-- +goose Up
-- Drop redundant index - the UNIQUE(workspace_id, year, month) constraint already creates an implicit index
DROP INDEX IF EXISTS idx_months_workspace_year_month;

-- +goose Down
CREATE INDEX idx_months_workspace_year_month ON months(workspace_id, year, month);
