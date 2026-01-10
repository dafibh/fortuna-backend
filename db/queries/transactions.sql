-- name: CreateTransaction :one
INSERT INTO transactions (
    workspace_id, account_id, name, amount, type,
    transaction_date, is_paid, cc_settlement_intent, notes, transfer_pair_id, category_id, is_cc_payment, template_id,
    cc_state, source, is_projected
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
-- When toggling paid status, also convert projections to actuals
-- Marking a projected transaction as paid acknowledges it as a real transaction
UPDATE transactions
SET is_paid = NOT is_paid, is_projected = false, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateTransactionSettlementIntent :one
UPDATE transactions
SET cc_settlement_intent = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL AND is_paid = false
RETURNING *;

-- name: UpdateTransaction :one
-- When editing any transaction, set is_projected = false to convert projections to actuals
-- The template_id is preserved so we know it originated from a recurring template
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
    is_projected = false,
    updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTransaction :execrows
-- When deleting any transaction (including projections), convert to actual first
-- This ensures deleted projections aren't recreated by the projection generator
UPDATE transactions
SET deleted_at = NOW(), is_projected = false, updated_at = NOW()
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

-- name: SumPaidIncomeByDateRange :one
-- Sum paid income within a date range for in-hand balance calculation
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) as total
FROM transactions
WHERE workspace_id = $1
  AND transaction_date >= $2
  AND transaction_date <= $3
  AND type = 'income'
  AND is_paid = true
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
    t.template_id,
    t.cc_state,
    t.billed_at,
    t.settled_at,
    t.source,
    t.is_projected,
    t.created_at,
    t.updated_at,
    t.deleted_at,
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

-- =====================================================
-- V2 CC LIFECYCLE QUERIES
-- =====================================================

-- name: UpdateCCState :one
-- Update CC transaction state (pending -> billed -> settled)
UPDATE transactions
SET
    cc_state = @cc_state,
    billed_at = CASE WHEN @cc_state = 'billed' THEN NOW() ELSE billed_at END,
    settled_at = CASE WHEN @cc_state = 'settled' THEN NOW() ELSE settled_at END,
    updated_at = NOW()
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
RETURNING *;

-- name: ToggleCCBilled :one
-- Toggle CC transaction between pending and billed states
UPDATE transactions
SET
    cc_state = CASE WHEN cc_state = 'pending' THEN 'billed' ELSE 'pending' END,
    billed_at = CASE WHEN cc_state = 'pending' THEN NOW() ELSE NULL END,
    updated_at = NOW()
WHERE workspace_id = @workspace_id AND id = @id AND deleted_at IS NULL
    AND cc_state IN ('pending', 'billed')
RETURNING *;

-- name: GetPendingCCByMonth :many
-- Get pending CC transactions for a specific month
SELECT t.*, a.name as account_name
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.cc_state = 'pending'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND EXTRACT(YEAR FROM t.transaction_date) = @year::INTEGER
    AND EXTRACT(MONTH FROM t.transaction_date) = @month::INTEGER
ORDER BY t.transaction_date DESC;

-- name: GetBilledCCByMonth :many
-- Get billed (unsettled, deferred) CC transactions for a specific month
SELECT t.*, a.name as account_name
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.cc_state = 'billed'
    AND t.cc_settlement_intent = 'deferred'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND EXTRACT(YEAR FROM t.transaction_date) = @year::INTEGER
    AND EXTRACT(MONTH FROM t.transaction_date) = @month::INTEGER
ORDER BY t.transaction_date DESC;

-- name: GetOverdueCC :many
-- Get CC transactions that are overdue (billed for 2+ months, still not settled)
SELECT t.*, a.name as account_name,
    EXTRACT(YEAR FROM t.transaction_date)::INTEGER as origin_year,
    EXTRACT(MONTH FROM t.transaction_date)::INTEGER as origin_month
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.cc_state = 'billed'
    AND t.cc_settlement_intent = 'deferred'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND t.billed_at < NOW() - INTERVAL '2 months'
ORDER BY t.transaction_date ASC;

-- name: GetDeferredCCByOriginMonth :many
-- Get deferred CC transactions from a specific origin month for settlement
SELECT t.*, a.name as account_name
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.cc_state = 'billed'
    AND t.cc_settlement_intent = 'deferred'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND EXTRACT(YEAR FROM t.transaction_date) = @year
    AND EXTRACT(MONTH FROM t.transaction_date) = @month
ORDER BY t.transaction_date DESC;

-- name: BulkSettleTransactions :execrows
-- Settle multiple CC transactions at once (used in settlement flow)
UPDATE transactions
SET
    cc_state = 'settled',
    settled_at = NOW(),
    is_paid = true,
    updated_at = NOW()
WHERE workspace_id = @workspace_id
    AND id = ANY(@ids::INTEGER[])
    AND cc_state = 'billed'
    AND deleted_at IS NULL;

-- name: GetTransactionsByIDs :many
-- Get multiple transactions by IDs for validation (settlement, etc.)
SELECT
    t.id, t.workspace_id, t.account_id, t.name, t.amount, t.type, t.transaction_date,
    t.is_paid, t.cc_settlement_intent, t.notes, t.category_id, t.transfer_pair_id,
    t.template_id, t.source, t.is_projected, t.cc_state, t.billed_at, t.settled_at,
    t.is_cc_payment, t.created_at, t.updated_at, t.deleted_at
FROM transactions t
WHERE t.workspace_id = @workspace_id
    AND t.id = ANY(@ids::INTEGER[])
    AND t.deleted_at IS NULL
ORDER BY t.transaction_date DESC;

-- name: GetDeferredCCByMonth :many
-- Get deferred CC transactions grouped by their origin month (for settlement view)
SELECT
    t.id, t.workspace_id, t.account_id, t.name, t.amount, t.type, t.transaction_date,
    t.is_paid, t.cc_settlement_intent, t.notes, t.category_id, t.transfer_pair_id,
    t.template_id, t.source, t.is_projected, t.cc_state, t.billed_at, t.settled_at,
    t.is_cc_payment, t.created_at, t.updated_at, t.deleted_at,
    a.name as account_name,
    EXTRACT(YEAR FROM t.transaction_date)::INTEGER as origin_year,
    EXTRACT(MONTH FROM t.transaction_date)::INTEGER as origin_month
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.type = 'expense'
    AND t.cc_state = 'billed'
    AND t.cc_settlement_intent = 'deferred'
    AND t.deleted_at IS NULL
ORDER BY t.transaction_date DESC;

-- name: GetCCMetricsByMonth :one
-- Get CC metrics for a specific month (purchases, outstanding, pending)
SELECT
    COALESCE(SUM(CASE WHEN t.cc_state IS NOT NULL THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as total_purchases,
    COALESCE(SUM(CASE WHEN t.cc_state = 'billed' AND t.cc_settlement_intent = 'deferred' THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as outstanding,
    COALESCE(SUM(CASE WHEN t.cc_state = 'pending' THEN t.amount ELSE 0 END), 0)::NUMERIC(12,2) as pending
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.type = 'expense'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND EXTRACT(YEAR FROM t.transaction_date) = @year::INTEGER
    AND EXTRACT(MONTH FROM t.transaction_date) = @month::INTEGER;

-- name: GetCCOutstandingTotal :one
-- Get total CC outstanding balance (all billed + deferred, not yet settled)
SELECT COALESCE(SUM(t.amount), 0)::NUMERIC(12,2) as total
FROM transactions t
JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND a.template = 'credit_card'
    AND t.cc_state = 'billed'
    AND t.cc_settlement_intent = 'deferred'
    AND t.type = 'expense'
    AND t.deleted_at IS NULL
    AND a.deleted_at IS NULL;

-- =====================================================
-- V2 PROJECTION QUERIES
-- =====================================================

-- name: GetTransactionsByMonth :many
-- Get all transactions for a specific month (including projections)
SELECT t.*, bc.name as category_name, a.name as account_name
FROM transactions t
LEFT JOIN budget_categories bc ON t.category_id = bc.id AND bc.deleted_at IS NULL
LEFT JOIN accounts a ON t.account_id = a.id
WHERE t.workspace_id = @workspace_id
    AND t.deleted_at IS NULL
    AND EXTRACT(YEAR FROM t.transaction_date) = @year
    AND EXTRACT(MONTH FROM t.transaction_date) = @month
ORDER BY t.transaction_date DESC, t.created_at DESC;

-- name: GetProjectionsByTemplateID :many
-- Get all projected transactions for a specific template
SELECT * FROM transactions
WHERE workspace_id = @workspace_id
    AND template_id = @template_id
    AND is_projected = true
    AND deleted_at IS NULL
ORDER BY transaction_date ASC;

-- name: DeleteProjectionsByTemplateID :execrows
-- Delete all projected transactions for a template (used when template is deleted)
UPDATE transactions
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = @workspace_id
    AND template_id = @template_id
    AND is_projected = true
    AND deleted_at IS NULL;

-- name: DeleteProjectionsBeyondDate :execrows
-- Delete projected transactions beyond a specific date (used when end_date is set)
UPDATE transactions
SET deleted_at = NOW(), updated_at = NOW()
WHERE workspace_id = @workspace_id
    AND template_id = @template_id
    AND is_projected = true
    AND transaction_date > @end_date
    AND deleted_at IS NULL;

-- name: OrphanActualsByTemplateID :execrows
-- Remove template_id from actual transactions when template is deleted
UPDATE transactions
SET template_id = NULL, updated_at = NOW()
WHERE workspace_id = @workspace_id
    AND template_id = @template_id
    AND is_projected = false
    AND deleted_at IS NULL;

-- name: CheckProjectionExists :one
-- Check if any transaction exists for a template in a specific month
-- This includes actual transactions (edited projections) and deleted ones
-- to prevent recreating projections that users have modified or deleted
SELECT COUNT(*)::INTEGER as count
FROM transactions
WHERE template_id = @template_id
    AND workspace_id = @workspace_id
    AND EXTRACT(YEAR FROM transaction_date) = @year::INTEGER
    AND EXTRACT(MONTH FROM transaction_date) = @month::INTEGER;

-- name: UpdateProjectedTransaction :one
-- Update a projected transaction (instance-level override)
UPDATE transactions
SET
    amount = COALESCE(sqlc.narg('amount'), amount),
    category_id = COALESCE(sqlc.narg('category_id'), category_id),
    name = COALESCE(sqlc.narg('name'), name),
    updated_at = NOW()
WHERE workspace_id = @workspace_id AND id = @id AND is_projected = true AND deleted_at IS NULL
RETURNING *;

-- name: UpdateTransactionAmount :exec
-- Update only the amount of a transaction (for overdue CC adjustments)
UPDATE transactions
SET amount = $3, updated_at = NOW()
WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: GetExpensesByDateRange :many
-- Get all expense transactions within a date range for future spending graph
SELECT
    t.id,
    t.workspace_id,
    t.account_id,
    t.name,
    t.amount,
    t.type,
    t.transaction_date,
    t.is_paid,
    t.category_id,
    t.cc_settlement_intent,
    t.notes,
    t.transfer_pair_id,
    t.is_cc_payment,
    t.template_id,
    t.cc_state,
    t.source,
    t.is_projected,
    t.billed_at,
    t.settled_at,
    t.created_at,
    t.updated_at
FROM transactions t
WHERE t.workspace_id = $1
    AND t.type = 'expense'
    AND t.transaction_date >= $2
    AND t.transaction_date < $3
    AND t.deleted_at IS NULL
ORDER BY t.transaction_date ASC;
