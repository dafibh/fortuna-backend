-- name: CreateLoan :one
INSERT INTO loans (
    workspace_id,
    provider_id,
    item_name,
    total_amount,
    num_months,
    purchase_date,
    interest_rate,
    monthly_payment,
    first_payment_year,
    first_payment_month,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetLoanByID :one
SELECT * FROM loans
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: ListLoans :many
SELECT * FROM loans
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListActiveLoans :many
SELECT l.* FROM loans l
WHERE l.workspace_id = $1
  AND l.deleted_at IS NULL
  AND (
    -- Loan is active if there are remaining payments
    -- Current month is before or equal to last payment month
    -- last_payment_year = first_payment_year + (num_months - 1) / 12
    -- last_payment_month = ((first_payment_month - 1 + num_months - 1) % 12) + 1
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12)) > $2
    OR (
      (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12)) = $2
      AND (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1) >= $3
    )
  )
ORDER BY l.created_at DESC;

-- name: ListCompletedLoans :many
SELECT l.* FROM loans l
WHERE l.workspace_id = $1
  AND l.deleted_at IS NULL
  AND (
    -- Loan is completed if current month is past the last payment month
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12)) < $2
    OR (
      (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12)) = $2
      AND (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1) < $3
    )
  )
ORDER BY l.created_at DESC;

-- name: UpdateLoan :one
UPDATE loans
SET item_name = $3,
    total_amount = $4,
    num_months = $5,
    purchase_date = $6,
    interest_rate = $7,
    monthly_payment = $8,
    first_payment_year = $9,
    first_payment_month = $10,
    notes = $11,
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteLoan :exec
UPDATE loans
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: CountActiveLoansByProvider :one
SELECT COUNT(*) FROM loans
WHERE provider_id = $1 AND workspace_id = $2 AND deleted_at IS NULL
  AND (
    (first_payment_year + ((first_payment_month - 1 + num_months - 1) / 12)) > $3
    OR (
      (first_payment_year + ((first_payment_month - 1 + num_months - 1) / 12)) = $3
      AND (((first_payment_month - 1 + num_months - 1) % 12) + 1) >= $4
    )
  );
