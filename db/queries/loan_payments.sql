-- name: CreateLoanPayment :one
INSERT INTO loan_payments (
    loan_id,
    payment_number,
    amount,
    due_year,
    due_month,
    paid,
    paid_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetLoanPaymentsByLoanID :many
SELECT * FROM loan_payments
WHERE loan_id = $1
ORDER BY payment_number;

-- name: GetLoanPaymentByID :one
SELECT * FROM loan_payments
WHERE id = $1;

-- name: GetLoanPaymentByLoanAndNumber :one
SELECT * FROM loan_payments
WHERE loan_id = $1 AND payment_number = $2;

-- name: UpdateLoanPaymentAmount :one
UPDATE loan_payments
SET amount = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ToggleLoanPaymentPaid :one
UPDATE loan_payments
SET paid = $2, paid_date = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetLoanPaymentsByMonth :many
SELECT lp.* FROM loan_payments lp
JOIN loans l ON lp.loan_id = l.id
WHERE l.workspace_id = $1
  AND lp.due_year = $2
  AND lp.due_month = $3
  AND l.deleted_at IS NULL
ORDER BY lp.due_year, lp.due_month, lp.payment_number;

-- name: GetUnpaidLoanPaymentsByMonth :many
SELECT lp.* FROM loan_payments lp
JOIN loans l ON lp.loan_id = l.id
WHERE l.workspace_id = $1
  AND lp.due_year = $2
  AND lp.due_month = $3
  AND lp.paid = FALSE
  AND l.deleted_at IS NULL
ORDER BY lp.due_year, lp.due_month, lp.payment_number;

-- name: GetLoanDeleteStats :one
SELECT
    COUNT(*)::INTEGER as total_count,
    COUNT(*) FILTER (WHERE paid = true)::INTEGER as paid_count,
    COUNT(*) FILTER (WHERE paid = false)::INTEGER as unpaid_count,
    COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total_amount
FROM loan_payments
WHERE loan_id = $1;

-- name: GetLoanPaymentsWithDetailsByMonth :many
SELECT
    lp.id,
    lp.loan_id,
    l.item_name,
    lp.payment_number,
    l.num_months as total_payments,
    lp.amount,
    lp.paid
FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND lp.due_year = $2
  AND lp.due_month = $3
  AND l.deleted_at IS NULL
ORDER BY l.item_name, lp.payment_number;

-- name: SumUnpaidLoanPaymentsByMonth :one
SELECT COALESCE(SUM(lp.amount), 0)::NUMERIC(12,2) as total
FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND lp.due_year = $2
  AND lp.due_month = $3
  AND lp.paid = FALSE
  AND l.deleted_at IS NULL;

-- name: GetEarliestUnpaidMonth :one
-- Returns the earliest (year, month) with unpaid payments for a provider
-- This is used for sequential enforcement in consolidated monthly payment mode
-- Handles gap months by finding the earliest unpaid month in ANY loan period
SELECT lp.due_year, lp.due_month
FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND l.provider_id = $2
  AND lp.paid = FALSE
  AND l.deleted_at IS NULL
ORDER BY lp.due_year ASC, lp.due_month ASC
LIMIT 1;

-- name: GetUnpaidPaymentsByProviderMonth :many
-- Returns all unpaid loan payments for a specific provider and month
-- Used for Pay Month action in consolidated monthly mode
SELECT lp.* FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND l.provider_id = $2
  AND lp.due_year = $3
  AND lp.due_month = $4
  AND lp.paid = FALSE
  AND l.deleted_at IS NULL
ORDER BY l.item_name, lp.payment_number;

-- name: BatchUpdatePaid :execrows
-- Atomically marks multiple loan payments as paid
-- Used for Pay Month action in consolidated monthly mode
-- Returns the number of rows affected
UPDATE loan_payments
SET paid = TRUE, paid_date = $2, updated_at = NOW()
WHERE id = ANY($1::int[])
  AND paid = FALSE;

-- name: GetPaymentsByIDs :many
-- Returns loan payments by their IDs with workspace validation via join
SELECT lp.* FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE lp.id = ANY($1::int[])
  AND l.workspace_id = $2
  AND l.deleted_at IS NULL;

-- name: SumPaymentAmountsByIDs :one
-- Returns the sum of amounts for given payment IDs
SELECT COALESCE(SUM(lp.amount), 0)::NUMERIC(12,2) as total
FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE lp.id = ANY($1::int[])
  AND l.workspace_id = $2
  AND l.deleted_at IS NULL;

-- name: GetTrendByMonth :many
-- Aggregates loan payments by year/month and provider for trend visualization
-- Returns monthly totals with provider breakdown and isPaid status
-- Gap months (no payments) are handled in the service layer
SELECT
  lp.due_year,
  lp.due_month,
  l.provider_id,
  lpr.name AS provider_name,
  COALESCE(SUM(lp.amount), 0)::NUMERIC(12,2) AS total,
  BOOL_AND(lp.paid) AS is_paid
FROM loan_payments lp
JOIN loans l ON lp.loan_id = l.id
JOIN loan_providers lpr ON l.provider_id = lpr.id
WHERE l.workspace_id = $1
  AND l.deleted_at IS NULL
  AND lpr.deleted_at IS NULL
  AND (
    (lp.due_year > $2) OR
    (lp.due_year = $2 AND lp.due_month >= $3)
  )
GROUP BY lp.due_year, lp.due_month, l.provider_id, lpr.name
ORDER BY lp.due_year, lp.due_month, l.provider_id;

-- name: GetLatestPaidMonth :one
-- Returns the latest (year, month) with paid payments for a provider
-- Used for reverse sequential enforcement in unpay action
SELECT lp.due_year, lp.due_month
FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND l.provider_id = $2
  AND lp.paid = TRUE
  AND l.deleted_at IS NULL
ORDER BY lp.due_year DESC, lp.due_month DESC
LIMIT 1;

-- name: GetPaidPaymentsByProviderMonth :many
-- Returns all paid loan payments for a specific provider and month
-- Used for Unpay Month action in consolidated monthly mode
SELECT lp.* FROM loan_payments lp
JOIN loans l ON l.id = lp.loan_id
WHERE l.workspace_id = $1
  AND l.provider_id = $2
  AND lp.due_year = $3
  AND lp.due_month = $4
  AND lp.paid = TRUE
  AND l.deleted_at IS NULL
ORDER BY l.item_name, lp.payment_number;

-- name: BatchUpdateUnpaid :execrows
-- Atomically marks multiple loan payments as unpaid
-- Used for Unpay Month action in consolidated monthly mode
-- Returns the number of rows affected
UPDATE loan_payments
SET paid = FALSE, paid_date = NULL, updated_at = NOW()
WHERE id = ANY($1::int[])
  AND paid = TRUE;
