//go:build integration

package wallet_test

import (
	"context"
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
