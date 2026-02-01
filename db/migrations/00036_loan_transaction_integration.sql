-- +goose Up
-- +goose StatementBegin
-- Consolidated Loans v2 Migration: Integrate loans with transactions
-- This migration drops the loan_payments table and adds loan_id to transactions,
-- making transactions the single source of truth for loan payments.
--
-- IMPORTANT: Existing loan_payments data will be lost. If production has
-- significant loan_payments data, coordinate a manual backup before running.

-- Step 1: Add loan_id column to transactions table
ALTER TABLE transactions ADD COLUMN loan_id INTEGER NULL;

-- Step 2: Add foreign key constraint with ON DELETE SET NULL
-- This ensures paid transactions are preserved (orphaned) when loans are deleted
ALTER TABLE transactions ADD CONSTRAINT fk_transactions_loan
    FOREIGN KEY (loan_id) REFERENCES loans(id) ON DELETE SET NULL;

-- Step 3: Create partial index for efficient loan-transaction lookups
-- Only index non-null values since most transactions won't be loan payments
CREATE INDEX idx_transactions_loan_id ON transactions(loan_id)
    WHERE loan_id IS NOT NULL;

-- Step 4: Drop loan_payments table (data will be lost - see note above)
DROP TABLE IF EXISTS loan_payments;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse the migration: recreate loan_payments and remove loan_id from transactions

-- Step 1: Recreate loan_payments table (structure only, data is not recoverable)
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

-- Step 2: Drop the index on loan_id (must drop before constraint)
DROP INDEX IF EXISTS idx_transactions_loan_id;

-- Step 3: Drop the foreign key constraint
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_loan;

-- Step 4: Drop the loan_id column from transactions
ALTER TABLE transactions DROP COLUMN IF EXISTS loan_id;
-- +goose StatementEnd
