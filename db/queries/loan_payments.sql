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
