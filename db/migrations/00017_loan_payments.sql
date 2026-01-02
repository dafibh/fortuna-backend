-- +goose Up
-- +goose StatementBegin
CREATE TABLE loan_payments (
    id SERIAL PRIMARY KEY,
    loan_id INTEGER NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    payment_number INTEGER NOT NULL CHECK (payment_number >= 1),
    amount NUMERIC(12,2) NOT NULL,
    due_year INTEGER NOT NULL,
    due_month INTEGER NOT NULL CHECK (due_month >= 1 AND due_month <= 12),
    paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(loan_id, payment_number)
);

CREATE INDEX idx_loan_payments_loan ON loan_payments(loan_id);
CREATE INDEX idx_loan_payments_due ON loan_payments(due_year, due_month);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS loan_payments;
-- +goose StatementEnd
