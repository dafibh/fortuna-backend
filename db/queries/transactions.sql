-- name: CreateTransaction :one
INSERT INTO transactions (
    workspace_id, account_id, name, amount, type,
    transaction_date, is_paid, cc_settlement_intent, notes, transfer_pair_id, category_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: GetTransactionsByWorkspace :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND deleted_at IS NULL
  AND ($2::INTEGER IS NULL OR account_id = $2)
  AND ($3::DATE IS NULL OR transaction_date >= $3)
  AND ($4::DATE IS NULL OR transaction_date <= $4)
  AND ($5::VARCHAR IS NULL OR type = $5)
ORDER BY transaction_date DESC, created_at DESC
LIMIT $6 OFFSET $7;

-- name: CountTransactionsByWorkspace :one
SELECT COUNT(*) FROM transactions
WHERE workspace_id = $1
  AND deleted_at IS NULL
  AND ($2::INTEGER IS NULL OR account_id = $2)
  AND ($3::DATE IS NULL OR transaction_date >= $3)
  AND ($4::DATE IS NULL OR transaction_date <= $4)
  AND ($5::VARCHAR IS NULL OR type = $5);

-- name: ToggleTransactionPaidStatus :one
UPDATE transactions
SET is_paid = NOT is_paid, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateTransactionSettlementIntent :one
UPDATE transactions
SET cc_settlement_intent = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL AND is_paid = false
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
SET
    name = $3,
    amount = $4,
    type = $5,
    transaction_date = $6,
    account_id = $7,
    cc_settlement_intent = $8,
    notes = $9,
    category_id = $10,
    updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTransaction :execrows
UPDATE transactions
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: SoftDeleteTransferPair :execrows
UPDATE transactions
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = $1 AND transfer_pair_id = $2 AND deleted_at IS NULL;

-- name: GetAccountTransactionSummaries :many
SELECT
    account_id,
    COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS sum_income,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS sum_expenses,
    COALESCE(SUM(CASE WHEN type = 'expense' AND is_paid = false THEN amount ELSE 0 END), 0) AS sum_unpaid_expenses
FROM transactions
WHERE workspace_id = $1 AND deleted_at IS NULL
GROUP BY account_id;

-- name: SumTransactionsByTypeAndDateRange :one
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = $4
  AND deleted_at IS NULL;

-- name: GetMonthlyTransactionSummaries :many
-- Batch query to get income/expense totals grouped by year/month for N+1 prevention
SELECT
    EXTRACT(YEAR FROM transaction_date)::INTEGER AS year,
    EXTRACT(MONTH FROM transaction_date)::INTEGER AS month,
    COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0)::NUMERIC(12,2) AS total_income,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)::NUMERIC(12,2) AS total_expenses
FROM transactions
WHERE workspace_id = $1
  AND deleted_at IS NULL
GROUP BY EXTRACT(YEAR FROM transaction_date), EXTRACT(MONTH FROM transaction_date)
ORDER BY year DESC, month DESC;

-- name: SumPaidExpensesByDateRange :one
-- Sum paid expenses within a date range for in-hand balance calculation
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = 'expense'
  AND is_paid = true
  AND deleted_at IS NULL;

-- name: SumUnpaidExpensesByDateRange :one
-- Sum unpaid expenses within a date range for disposable income calculation
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = 'expense'
  AND is_paid = false
  AND deleted_at IS NULL;

-- name: GetCCPayableSummary :many
-- Get unpaid CC transaction totals grouped by settlement intent
SELECT
    cc_settlement_intent,
    COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND a.deleted_at IS NULL
  AND t.type = 'expense'
  AND t.is_paid = false
  AND t.deleted_at IS NULL
  AND t.cc_settlement_intent IS NOT NULL
GROUP BY cc_settlement_intent;

-- name: GetTransactionsWithCategory :many
-- Returns transactions with category name joined for display
SELECT
    t.id,
    t.workspace_id,
    t.account_id,
    t.name,
    t.amount,
    t.type,
    t.transaction_date,
    t.is_paid,
    t.cc_settlement_intent,
    t.notes,
    t.transfer_pair_id,
    t.category_id,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    bc.name AS category_name
FROM transactions t
LEFT JOIN budget_categories bc ON t.category_id = bc.id AND bc.deleted_at IS NULL
WHERE t.workspace_id = $1
  AND t.deleted_at IS NULL
  AND ($2::INTEGER IS NULL OR t.account_id = $2)
  AND ($3::DATE IS NULL OR t.transaction_date >= $3)
  AND ($4::DATE IS NULL OR t.transaction_date <= $4)
  AND ($5::VARCHAR IS NULL OR t.type = $5)
ORDER BY t.transaction_date DESC, t.created_at DESC
LIMIT $6 OFFSET $7;

-- name: GetRecentlyUsedCategories :many
-- Returns recently used categories for suggestions dropdown
SELECT DISTINCT
    bc.id,
    bc.name,
    MAX(t.created_at) AS last_used
FROM transactions t
JOIN budget_categories bc ON t.category_id = bc.id AND bc.deleted_at IS NULL
WHERE t.workspace_id = $1
  AND t.deleted_at IS NULL
  AND t.category_id IS NOT NULL
GROUP BY bc.id, bc.name
ORDER BY last_used DESC
LIMIT 5;
