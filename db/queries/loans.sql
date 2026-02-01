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
    account_id,
    settlement_intent,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
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

-- name: UpdateLoanPartial :one
-- Only updates editable fields (item_name, notes) - amount/months/dates are locked after creation
UPDATE loans
SET item_name = $3,
    notes = $4,
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateLoanEditableFields :one
-- Updates editable fields with optional provider change
-- Provider can only change if no payments have been made (validated at service layer)
UPDATE loans
SET item_name = $3,
    provider_id = $4,
    notes = $5,
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

-- CL v2: Use transactions with loan_id instead of loan_payments table

-- name: GetLoansWithStats :many
-- Get all loans with payment stats calculated from transactions
SELECT
    l.id,
    l.workspace_id,
    l.provider_id,
    l.item_name,
    l.total_amount,
    l.num_months,
    l.purchase_date,
    l.interest_rate,
    l.monthly_payment,
    l.first_payment_year,
    l.first_payment_month,
    l.account_id,
    l.settlement_intent,
    l.notes,
    l.created_at,
    l.updated_at,
    l.deleted_at,
    -- Calculated last payment month/year
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12))::INTEGER as last_payment_year,
    (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1)::INTEGER as last_payment_month,
    -- Payment stats from transactions
    COUNT(t.id)::INTEGER as total_count,
    COUNT(t.id) FILTER (WHERE t.is_paid = true)::INTEGER as paid_count,
    COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0)::NUMERIC(12,2) as remaining_balance
FROM loans l
LEFT JOIN transactions t ON t.loan_id = l.id AND t.deleted_at IS NULL
WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
GROUP BY l.id
ORDER BY l.created_at DESC;

-- name: GetActiveLoansWithStats :many
-- Get active loans (with remaining balance) with payment stats calculated from transactions
SELECT
    l.id,
    l.workspace_id,
    l.provider_id,
    l.item_name,
    l.total_amount,
    l.num_months,
    l.purchase_date,
    l.interest_rate,
    l.monthly_payment,
    l.first_payment_year,
    l.first_payment_month,
    l.account_id,
    l.settlement_intent,
    l.notes,
    l.created_at,
    l.updated_at,
    l.deleted_at,
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12))::INTEGER as last_payment_year,
    (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1)::INTEGER as last_payment_month,
    COUNT(t.id)::INTEGER as total_count,
    COUNT(t.id) FILTER (WHERE t.is_paid = true)::INTEGER as paid_count,
    COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0)::NUMERIC(12,2) as remaining_balance
FROM loans l
LEFT JOIN transactions t ON t.loan_id = l.id AND t.deleted_at IS NULL
WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
GROUP BY l.id
HAVING COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0) > 0
ORDER BY l.created_at DESC;

-- name: GetCompletedLoansWithStats :many
-- Get completed loans (no remaining balance) with payment stats calculated from transactions
SELECT
    l.id,
    l.workspace_id,
    l.provider_id,
    l.item_name,
    l.total_amount,
    l.num_months,
    l.purchase_date,
    l.interest_rate,
    l.monthly_payment,
    l.first_payment_year,
    l.first_payment_month,
    l.account_id,
    l.settlement_intent,
    l.notes,
    l.created_at,
    l.updated_at,
    l.deleted_at,
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12))::INTEGER as last_payment_year,
    (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1)::INTEGER as last_payment_month,
    COUNT(t.id)::INTEGER as total_count,
    COUNT(t.id) FILTER (WHERE t.is_paid = true)::INTEGER as paid_count,
    COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0)::NUMERIC(12,2) as remaining_balance
FROM loans l
LEFT JOIN transactions t ON t.loan_id = l.id AND t.deleted_at IS NULL
WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
GROUP BY l.id
HAVING COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0) = 0
ORDER BY l.created_at DESC;

-- name: GetLoansWithStatsByProvider :many
-- Get all loans for a specific provider with payment stats calculated from transactions
-- Orders unpaid items first, then by item name
SELECT
    l.id,
    l.workspace_id,
    l.provider_id,
    l.item_name,
    l.total_amount,
    l.num_months,
    l.purchase_date,
    l.interest_rate,
    l.monthly_payment,
    l.first_payment_year,
    l.first_payment_month,
    l.account_id,
    l.settlement_intent,
    l.notes,
    l.created_at,
    l.updated_at,
    l.deleted_at,
    (l.first_payment_year + ((l.first_payment_month - 1 + l.num_months - 1) / 12))::INTEGER as last_payment_year,
    (((l.first_payment_month - 1 + l.num_months - 1) % 12) + 1)::INTEGER as last_payment_month,
    COUNT(t.id)::INTEGER as total_count,
    COUNT(t.id) FILTER (WHERE t.is_paid = true)::INTEGER as paid_count,
    COALESCE(SUM(t.amount) FILTER (WHERE t.is_paid = false), 0)::NUMERIC(12,2) as remaining_balance
FROM loans l
LEFT JOIN transactions t ON t.loan_id = l.id AND t.deleted_at IS NULL
WHERE l.workspace_id = $1 AND l.provider_id = $2 AND l.deleted_at IS NULL
GROUP BY l.id
ORDER BY (COUNT(t.id) FILTER (WHERE t.is_paid = false) > 0) DESC, l.item_name ASC;
