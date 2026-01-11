-- name: CreateTransaction :one
INSERT INTO transactions (
    workspace_id, account_id, name, amount, type,
    transaction_date, is_paid, notes, transfer_pair_id, category_id, is_cc_payment,
    billed_at, settlement_intent,
    source, template_id, is_projected
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
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
WHERE workspace_id = @workspace_id
  AND deleted_at IS NULL
  AND (sqlc.narg('account_id')::INTEGER IS NULL OR account_id = sqlc.narg('account_id'))
  AND (sqlc.narg('start_date')::DATE IS NULL OR transaction_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::DATE IS NULL OR transaction_date <= sqlc.narg('end_date'))
  AND (sqlc.narg('type')::VARCHAR IS NULL OR type = sqlc.narg('type'));

-- name: ToggleTransactionPaidStatus :one
UPDATE transactions
SET is_paid = NOT is_paid, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
SET
    name = $3,
    amount = $4,
    type = $5,
    transaction_date = $6,
    account_id = $7,
    notes = $8,
    category_id = $9,
    is_paid = $10,
    billed_at = $11,
    settlement_intent = $12,
    source = $13,
    template_id = $14,
    is_projected = $15,
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
-- For regular accounts: only count paid transactions
-- For CC accounts: count all expenses (isPaid means settled, not whether purchase happened)
SELECT
    account_id,
    COALESCE(SUM(CASE WHEN type = 'income' AND is_paid = true THEN amount ELSE 0 END), 0) AS sum_income,
    COALESCE(SUM(CASE WHEN type = 'expense' AND is_paid = true THEN amount ELSE 0 END), 0) AS sum_expenses,
    COALESCE(SUM(CASE WHEN type = 'expense' AND is_paid = false THEN amount ELSE 0 END), 0) AS sum_unpaid_expenses,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS sum_all_expenses
FROM transactions
WHERE workspace_id = $1 AND deleted_at IS NULL
GROUP BY account_id;

-- name: SumTransactionsByTypeAndDateRange :one
-- Only count paid transactions
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = $4
  AND is_paid = true
  AND deleted_at IS NULL;

-- name: GetMonthlyTransactionSummaries :many
-- Batch query to get income/expense totals grouped by year/month for N+1 prevention
-- Only count paid transactions
SELECT
    EXTRACT(YEAR FROM transaction_date)::INTEGER AS year,
    EXTRACT(MONTH FROM transaction_date)::INTEGER AS month,
    COALESCE(SUM(CASE WHEN type = 'income' AND is_paid = true THEN amount ELSE 0 END), 0)::NUMERIC(12,2) AS total_income,
    COALESCE(SUM(CASE WHEN type = 'expense' AND is_paid = true THEN amount ELSE 0 END), 0)::NUMERIC(12,2) AS total_expenses
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
-- Sum unpaid expenses within a date range (ALL unpaid, including deferred CC)
-- Used for balance calculations where all obligations matter
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = 'expense'
  AND is_paid = false
  AND deleted_at IS NULL;

-- name: SumUnpaidExpensesForDisposable :one
-- Sum unpaid expenses for disposable income calculation
-- EXCLUDES deferred CC transactions (those are for next month)
-- Includes: non-CC unpaid expenses + immediate CC expenses
SELECT COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total
FROM transactions t
LEFT JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND t.transaction_date >= $2
  AND t.transaction_date <= $3
  AND t.type = 'expense'
  AND t.is_paid = false
  AND t.deleted_at IS NULL
  AND NOT (a.template = 'credit_card' AND t.settlement_intent = 'deferred');

-- name: SumDeferredCCByDateRange :one
-- Sum deferred CC expenses within a date range
-- Used for next month projections
SELECT COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND t.transaction_date >= $2
  AND t.transaction_date <= $3
  AND t.type = 'expense'
  AND t.is_paid = false
  AND t.settlement_intent = 'deferred'
  AND a.template = 'credit_card'
  AND t.deleted_at IS NULL;

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
    t.notes,
    t.transfer_pair_id,
    t.category_id,
    t.is_cc_payment,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    t.billed_at,
    t.settlement_intent,
    t.source,
    t.template_id,
    t.is_projected,
    bc.name AS category_name
FROM transactions t
LEFT JOIN budget_categories bc ON t.category_id = bc.id AND bc.deleted_at IS NULL
WHERE t.workspace_id = @workspace_id
  AND t.deleted_at IS NULL
  AND (sqlc.narg('account_id')::INTEGER IS NULL OR t.account_id = sqlc.narg('account_id'))
  AND (sqlc.narg('start_date')::DATE IS NULL OR t.transaction_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::DATE IS NULL OR t.transaction_date <= sqlc.narg('end_date'))
  AND (sqlc.narg('type')::VARCHAR IS NULL OR t.type = sqlc.narg('type'))
ORDER BY t.transaction_date DESC, t.created_at DESC
LIMIT @page_size OFFSET @page_offset;

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

-- name: GetTransactionsByIDs :many
-- Get multiple transactions by their IDs
SELECT * FROM transactions
WHERE workspace_id = $1
  AND id = ANY($2::int[])
  AND deleted_at IS NULL
ORDER BY id;

-- ========================================
-- CC Lifecycle Operations (v2 - Simplified)
-- CC State: pending (billed_at IS NULL), billed (billed_at IS NOT NULL, is_paid=false), settled (is_paid=true)
-- ========================================

-- name: ToggleBilledStatus :one
-- Toggle billed status for a CC transaction (pending <-> billed)
UPDATE transactions
SET billed_at = CASE WHEN billed_at IS NULL THEN NOW() ELSE NULL END,
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetPendingCCByMonth :many
-- Get pending CC transactions (billed_at IS NULL) for a specific month range
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NULL
  AND t.is_paid = false
  AND t.transaction_date >= $2 AND t.transaction_date < $3
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date DESC;

-- name: GetBilledCCByMonth :many
-- Get billed CC transactions with deferred settlement intent for a month range
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NOT NULL
  AND t.is_paid = false
  AND t.settlement_intent = 'deferred'
  AND t.transaction_date >= $2 AND t.transaction_date < $3
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date DESC;

-- name: GetOverdueCC :many
-- Get CC transactions that are billed but overdue (2+ months old)
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NOT NULL
  AND t.is_paid = false
  AND t.settlement_intent = 'deferred'
  AND t.billed_at < NOW() - INTERVAL '2 months'
  AND t.deleted_at IS NULL
ORDER BY t.billed_at ASC;

-- name: BulkSettleTransactions :many
-- Bulk update multiple transactions to settled state (is_paid = true)
UPDATE transactions
SET is_paid = true,
    updated_at = NOW()
WHERE id = ANY($1::int[])
  AND workspace_id = $2
  AND billed_at IS NOT NULL
  AND is_paid = false
  AND deleted_at IS NULL
RETURNING *;

-- name: BatchToggleToBilled :many
-- Batch toggle multiple transactions from pending to billed
UPDATE transactions
SET billed_at = NOW(),
    updated_at = NOW()
WHERE id = ANY($1::int[])
  AND workspace_id = $2
  AND billed_at IS NULL
  AND deleted_at IS NULL
RETURNING *;

-- name: GetCCMetrics :one
-- Get CC metrics (pending, outstanding, purchases) for a month range
-- Simplified: pending = billed_at IS NULL, billed = billed_at IS NOT NULL AND is_paid = false, settled = is_paid = true
-- purchases = CC expenses this month (EXCLUDES deferred - those are next month's obligations)
-- pending = not yet billed transactions (current month + deferred from previous months)
-- outstanding = billed transactions to settle this month:
--   1. deferred intent from previous months (billed)
--   2. immediate intent from current month (billed)
SELECT
    COALESCE(SUM(CASE WHEN t.billed_at IS NULL AND t.is_paid = false AND (
        (t.transaction_date >= $2 AND t.transaction_date < $3) OR
        (t.settlement_intent = 'deferred' AND t.transaction_date < $2)
    ) THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as pending_total,
    COALESCE(SUM(CASE WHEN t.billed_at IS NOT NULL AND t.is_paid = false AND (
        (t.settlement_intent = 'deferred' AND t.transaction_date < $2) OR
        (t.settlement_intent = 'immediate' AND t.transaction_date >= $2 AND t.transaction_date < $3)
    ) THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as outstanding_total,
    COALESCE(SUM(CASE WHEN t.type = 'expense' AND t.transaction_date >= $2 AND t.transaction_date < $3 AND COALESCE(t.settlement_intent, 'immediate') != 'deferred' THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as purchases_total
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND (
    (t.transaction_date >= $2 AND t.transaction_date < $3) OR
    (t.is_paid = false AND t.settlement_intent = 'deferred' AND t.transaction_date < $2)
  )
  AND t.deleted_at IS NULL;

-- ========================================
-- Projection Management (v2)
-- ========================================

-- name: GetProjectionsByTemplate :many
-- Get all projected transactions for a specific template
SELECT * FROM transactions
WHERE workspace_id = $1
  AND template_id = $2
  AND is_projected = true
  AND deleted_at IS NULL
ORDER BY transaction_date;

-- name: DeleteProjectionsByTemplate :exec
-- Delete all projected transactions for a template (used when deleting template)
DELETE FROM transactions
WHERE workspace_id = $1
  AND template_id = $2
  AND is_projected = true;

-- name: DeleteProjectionsBeyondDate :exec
-- Delete projections beyond a specific date (used when changing template end_date)
DELETE FROM transactions
WHERE workspace_id = $1
  AND template_id = $2
  AND is_projected = true
  AND transaction_date > $3;

-- name: OrphanActualsByTemplate :exec
-- Unlink actual transactions from template (keep them, clear template_id)
UPDATE transactions
SET template_id = NULL,
    updated_at = NOW()
WHERE workspace_id = $1
  AND template_id = $2
  AND is_projected = false
  AND deleted_at IS NULL;

-- name: GetDeferredForSettlement :many
-- Get all billed, deferred transactions that need settlement (ordered by date)
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NOT NULL
  AND t.is_paid = false
  AND t.settlement_intent = 'deferred'
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date ASC;

-- name: GetImmediateForSettlement :many
-- Get billed transactions with immediate intent for the current month
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NOT NULL
  AND t.is_paid = false
  AND t.settlement_intent = 'immediate'
  AND t.transaction_date >= $2
  AND t.transaction_date < $3
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date ASC;

-- name: GetPendingDeferredCC :many
-- Get pending (not yet billed) deferred CC transactions for visibility
-- These are transactions that will need to be paid next month once billed
SELECT * FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
  AND a.template = 'credit_card'
  AND t.billed_at IS NULL
  AND t.is_paid = false
  AND t.settlement_intent = 'deferred'
  AND t.transaction_date >= $2
  AND t.transaction_date < $3
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date ASC;

-- name: GetTransactionsForAggregation :many
-- Returns all transactions in a date range with category name for aggregation (no pagination)
-- Used by dashboard future spending calculations
SELECT
    t.id,
    t.workspace_id,
    t.account_id,
    t.name,
    t.amount,
    t.type,
    t.transaction_date,
    t.is_paid,
    t.notes,
    t.transfer_pair_id,
    t.category_id,
    t.is_cc_payment,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    t.billed_at,
    t.settlement_intent,
    t.source,
    t.template_id,
    t.is_projected,
    bc.name AS category_name
FROM transactions t
LEFT JOIN budget_categories bc ON t.category_id = bc.id AND bc.deleted_at IS NULL
WHERE t.workspace_id = @workspace_id
  AND t.deleted_at IS NULL
  AND t.transaction_date >= @start_date::DATE
  AND t.transaction_date <= @end_date::DATE
ORDER BY t.transaction_date DESC, t.created_at DESC;
