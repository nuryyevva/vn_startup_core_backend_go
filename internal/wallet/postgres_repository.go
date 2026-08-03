package wallet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/db/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: sqlc.New(pool)}
}

func (r *PostgresRepository) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	balance, err := r.queries.GetWalletBalance(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get wallet balance: %w", err)
	}
	return balance, nil
}

func (r *PostgresRepository) Credit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error) {
	return r.createTransaction(ctx, userID, amount, reason, false)
}

func (r *PostgresRepository) Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error) {
	return r.createTransaction(ctx, userID, -amount, reason, true)
}

// createTransaction locks the user's wallet for the duration of the
// transaction (see LockWallet's doc comment for why an advisory lock is
// used instead of SELECT ... FOR UPDATE), optionally verifies the
// resulting balance won't go negative, and inserts the ledger row.
func (r *PostgresRepository) createTransaction(ctx context.Context, userID uuid.UUID, amount int32, reason string, checkSufficient bool) (Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.queries.WithTx(tx)

	if err := q.LockWallet(ctx, userID.String()); err != nil {
		return Transaction{}, fmt.Errorf("lock wallet: %w", err)
	}

	if checkSufficient {
		balance, err := q.GetWalletBalance(ctx, userID)
		if err != nil {
			return Transaction{}, fmt.Errorf("get wallet balance: %w", err)
		}
		if balance+int64(amount) < 0 {
			return Transaction{}, ErrInsufficientFunds
		}
	}

	dbTx, err := q.CreateWalletTransaction(ctx, sqlc.CreateWalletTransactionParams{
		UserID: userID,
		Amount: amount,
		Reason: reason,
	})
	if err != nil {
		return Transaction{}, fmt.Errorf("insert wallet transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit tx: %w", err)
	}

	return toDomainTransaction(dbTx), nil
}

func toDomainTransaction(t sqlc.WalletTransaction) Transaction {
	return Transaction{
		ID:        t.ID,
		UserID:    t.UserID,
		Amount:    t.Amount,
		Reason:    t.Reason,
		CreatedAt: t.CreatedAt,
	}
}
