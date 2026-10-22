//go:build integration

package wallet_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/internal/testutil"
	"vn_startup_core_backend_go/internal/wallet"
)

func TestWalletIntegration_GrantDebitBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "wallet-player@example.com", "hash")
	require.NoError(t, err)

	repo := wallet.NewPostgresRepository(pool)
	svc := wallet.NewService(repo)

	balance, err := svc.Balance(ctx, player.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, balance)

	_, err = svc.Grant(ctx, player.ID, 100)
	require.NoError(t, err)

	balance, err = svc.Balance(ctx, player.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 100, balance)

	require.NoError(t, svc.Debit(ctx, player.ID, 30, wallet.ReasonChoicePurchase))

	balance, err = svc.Balance(ctx, player.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 70, balance)

	err = svc.Debit(ctx, player.ID, 1000, wallet.ReasonChoicePurchase)
	require.Error(t, err)

	// Balance must be unchanged after a rejected debit.
	balance, err = svc.Balance(ctx, player.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 70, balance)
}

// TestWalletIntegration_ConcurrentDebitsNeverGoNegative fires many
// concurrent debits against a wallet that can only afford a handful of
// them, and asserts the ledger-derived balance never drops below zero —
// the guarantee the advisory-lock-per-user design in
// PostgresRepository.createTransaction is meant to provide.
func TestWalletIntegration_ConcurrentDebitsNeverGoNegative(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	authRepo := auth.NewPostgresRepository(pool)
	player, err := authRepo.CreateUser(ctx, "wallet-concurrent@example.com", "hash")
	require.NoError(t, err)

	repo := wallet.NewPostgresRepository(pool)
	svc := wallet.NewService(repo)

	const startingBalance = 100
	const debitAmount = 15
	const attempts = 20 // 20*15 = 300, far more than the 100 available

	_, err = svc.Grant(ctx, player.ID, startingBalance)
	require.NoError(t, err)

	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.Debit(ctx, player.ID, debitAmount, wallet.ReasonChoicePurchase); err != nil {
				atomic.AddInt64(&failureCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	balance, err := svc.Balance(ctx, player.ID)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, balance, int64(0), "balance must never go negative")
	assert.Equal(t, int64(startingBalance)-successCount*debitAmount, balance)
	assert.Greater(t, failureCount, int64(0), "expected at least one debit to be rejected for insufficient funds")
	assert.LessOrEqual(t, successCount, int64(startingBalance/debitAmount))
}

// TestWalletIntegration_StartingBalanceMigration replays migration 0021 on
// accounts in every state it can meet and checks each ends at exactly the
// starting balance, without the correction counting as spending.
func TestWalletIntegration_StartingBalanceMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()
	authRepo := auth.NewPostgresRepository(pool)
	svc := wallet.NewService(wallet.NewPostgresRepository(pool))

	empty, err := authRepo.CreateUser(ctx, "empty@example.com", "hash")
	require.NoError(t, err)
	rich, err := authRepo.CreateUser(ctx, "rich@example.com", "hash")
	require.NoError(t, err)
	spender, err := authRepo.CreateUser(ctx, "spender@example.com", "hash")
	require.NoError(t, err)
	exact, err := authRepo.CreateUser(ctx, "exact@example.com", "hash")
	require.NoError(t, err)

	_, err = svc.Grant(ctx, rich.ID, 31175)
	require.NoError(t, err)
	_, err = svc.Grant(ctx, spender.ID, 1000)
	require.NoError(t, err)
	require.NoError(t, svc.Debit(ctx, spender.ID, 900, wallet.ReasonDialogMessage))
	_, err = svc.Grant(ctx, exact.ID, wallet.StartingBalance)
	require.NoError(t, err)

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "0021_wallet_starting_balance.up.sql"))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(migration))
	require.NoError(t, err)

	for _, u := range []auth.User{empty, rich, spender, exact} {
		balance, err := svc.Balance(ctx, u.ID)
		require.NoError(t, err)
		assert.EqualValues(t, wallet.StartingBalance, balance, u.Email)
	}

	spent, err := svc.TotalSpent(ctx, rich.ID)
	require.NoError(t, err)
	assert.Zero(t, spent, "taking the extra diamonds away isn't spending")
	spent, err = svc.TotalSpent(ctx, spender.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 900, spent)

	var adjustments int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM wallet_transactions WHERE user_id = $1 AND reason = 'balance_adjustment'`, exact.ID).Scan(&adjustments))
	assert.Zero(t, adjustments, "an account already at the starting balance is left alone")
}
