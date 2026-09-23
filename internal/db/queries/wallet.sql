-- name: LockWallet :exec
-- Advisory transaction-scoped lock keyed by user id. wallet_transactions has
-- no balance row to lock with SELECT ... FOR UPDATE, and locking existing
-- rows wouldn't block a concurrent INSERT anyway, so we serialize
-- read-balance-then-insert per user with an advisory lock instead.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg('user_id')::text));

-- name: GetWalletBalance :one
SELECT COALESCE(SUM(amount), 0)::bigint AS balance
FROM wallet_transactions
WHERE user_id = $1;

-- name: CreateWalletTransaction :one
INSERT INTO wallet_transactions (user_id, amount, reason)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTotalDiamondsSpent :one
SELECT COALESCE(-SUM(amount), 0)::bigint AS total_spent
FROM wallet_transactions
WHERE user_id = $1 AND amount < 0;
