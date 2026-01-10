-- name: CreateTransaction :one
INSERT INTO transactions (
    workspace_id, account_id, name, amount, type,
    transaction_date, is_paid, cc_settlement_intent, notes, transfer_pair_id, category_id, is_cc_payment, recurring_transaction_id,
    cc_state, billed_at, settled_at, settlement_intent,
    source, template_id, is_projected
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
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
    cc_state = $11,
    billed_at = $12,
    settled_at = $13,
    settlement_intent = $14,
    source = $15,
    template_id = $16,
    is_projected = $17,
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
    t.is_cc_payment,
    t.recurring_transaction_id,
    t.created_at,
    t.updated_at,
    t.deleted_at,
    t.cc_state,
    t.billed_at,
    t.settled_at,
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

-- name: GetCCPayableBreakdown :many
-- Get all unpaid CC transactions with settlement intent for payable breakdown
SELECT
    t.id,
    t.name,
    t.amount,
    t.transaction_date,
    t.cc_settlement_intent,
    t.account_id,
    a.name as account_name
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = $1
    AND a.template = 'credit_card'
    AND t.type = 'expense'
    AND t.is_paid = false
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
ORDER BY t.cc_settlement_intent, a.name, t.transaction_date DESC;

-- ========================================
-- CC Lifecycle Operations (v2)
-- ========================================

-- name: UpdateCCState :one
-- Update CC state and timestamps for a transaction
UPDATE transactions
SET cc_state = $3,
    billed_at = $4,
    settled_at = $5,
    updated_at = NOW()
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetPendingCCByMonth :many
-- Get pending CC transactions for a specific month range
SELECT * FROM transactions
WHERE workspace_id = $1
  AND cc_state = 'pending'
  AND transaction_date >= $2 AND transaction_date < $3
  AND deleted_at IS NULL
ORDER BY transaction_date DESC;

-- name: GetBilledCCByMonth :many
-- Get billed CC transactions with deferred settlement intent for a month range
SELECT * FROM transactions
WHERE workspace_id = $1
  AND cc_state = 'billed'
  AND settlement_intent = 'deferred'
  AND transaction_date >= $2 AND transaction_date < $3
  AND deleted_at IS NULL
ORDER BY transaction_date DESC;

-- name: GetOverdueCC :many
-- Get CC transactions that are billed but overdue (2+ months old)
SELECT * FROM transactions
WHERE workspace_id = $1
  AND cc_state = 'billed'
  AND settlement_intent = 'deferred'
  AND billed_at < NOW() - INTERVAL '2 months'
  AND deleted_at IS NULL
ORDER BY billed_at ASC;

-- name: BulkSettleTransactions :many
-- Bulk update multiple transactions to settled state
UPDATE transactions
SET cc_state = 'settled',
    settled_at = NOW(),
    updated_at = NOW()
WHERE id = ANY($1::int[])
  AND workspace_id = $2
  AND cc_state = 'billed'
  AND deleted_at IS NULL
RETURNING *;

-- name: GetCCMetrics :one
-- Get CC metrics (pending, billed, total) for a month range
-- month_total = pending + billed (excludes settled transactions)
SELECT
    COALESCE(SUM(CASE WHEN cc_state = 'pending' THEN amount ELSE 0 END), 0)::NUMERIC(12,2) as pending_total,
    COALESCE(SUM(CASE WHEN cc_state = 'billed' AND settlement_intent = 'deferred' THEN amount ELSE 0 END), 0)::NUMERIC(12,2) as billed_total,
    COALESCE(SUM(CASE WHEN cc_state IN ('pending', 'billed') THEN amount ELSE 0 END), 0)::NUMERIC(12,2) as month_total
FROM transactions
WHERE workspace_id = $1
  AND cc_state IS NOT NULL
  AND transaction_date >= $2 AND transaction_date < $3
  AND deleted_at IS NULL;

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
