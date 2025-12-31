-- +goose Up
CREATE TABLE months (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    year INTEGER NOT NULL CHECK (year >= 2000 AND year <= 2100),
    month INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    starting_balance NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, year, month)
);

CREATE INDEX idx_months_workspace_year_month ON months(workspace_id, year, month);

-- +goose Down
DROP TABLE IF EXISTS months;
