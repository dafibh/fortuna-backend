-- +goose Up

CREATE TABLE budget_allocations (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
    category_id INTEGER NOT NULL REFERENCES budget_categories(id),
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_budget_allocation UNIQUE (workspace_id, category_id, year, month)
);

CREATE INDEX idx_budget_allocations_month ON budget_allocations(workspace_id, year, month);

-- +goose Down

DROP INDEX IF EXISTS idx_budget_allocations_month;
DROP TABLE IF EXISTS budget_allocations;
