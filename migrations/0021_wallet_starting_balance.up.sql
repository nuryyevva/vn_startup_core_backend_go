-- Every account starts with 600 diamonds (wallet.StartingBalance) and, for
-- now, can't get more. Accounts that existed before this rule get the same
-- deal: one adjustment entry each brings the balance to exactly 600. The
-- ledger keeps its history; adjustments are corrections, not spending.
INSERT INTO wallet_transactions (user_id, amount, reason)
SELECT u.id, (600 - COALESCE(SUM(t.amount), 0))::int, 'balance_adjustment'
FROM users u
LEFT JOIN wallet_transactions t ON t.user_id = u.id
GROUP BY u.id
HAVING 600 - COALESCE(SUM(t.amount), 0) <> 0;
