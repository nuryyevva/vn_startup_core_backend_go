package wallet

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

// Repository persists wallet_transactions. There is no stored balance
// column anywhere: the balance is always the SUM of a user's transactions,
// computed on the fly, so it can never drift out of sync with the ledger.
type Repository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	// Credit records a positive-amount transaction (e.g. a dev grant).
	Credit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error)
	// Debit records a negative-amount transaction after verifying the
	// user's balance covers it, serialized per-user so concurrent debits
	// can never push the balance negative.
	Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error)
}
