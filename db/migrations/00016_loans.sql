-- +goose Up
-- +goose StatementBegin
CREATE TABLE loans (
    id SERIAL PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
    provider_id INTEGER NOT NULL REFERENCES loan_providers(id),
    item_name VARCHAR(200) NOT NULL,
    total_amount NUMERIC(12,2) NOT NULL,
    num_months INTEGER NOT NULL CHECK (num_months >= 1),
    purchase_date DATE NOT NULL,
    interest_rate NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    monthly_payment NUMERIC(12,2) NOT NULL,
    first_payment_year INTEGER NOT NULL,
    first_payment_month INTEGER NOT NULL CHECK (first_payment_month >= 1 AND first_payment_month <= 12),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_loans_workspace ON loans(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_loans_provider ON loans(provider_id) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_loans_provider;
DROP INDEX IF EXISTS idx_loans_workspace;
DROP TABLE IF EXISTS loans;
-- +goose StatementEnd
