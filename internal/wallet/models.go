package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Amount    int32
	Reason    string
	CreatedAt time.Time
}

const (
	ReasonDevGrant       = "dev_grant"
	ReasonChoicePurchase = "choice_purchase"
	ReasonDialogMessage  = "dialog_message"
	ReasonSignupBonus    = "signup_bonus"
	// ReasonBalanceAdjustment brought every account that existed before the
	// starting-balance rule to StartingBalance (migration 0021). It's a
	// correction, not spending, so it never counts toward "diamonds spent".
	ReasonBalanceAdjustment = "balance_adjustment"
)

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

type GrantRequest struct {
	Amount int32 `json:"amount"`
}

type TransactionResponse struct {
	ID        uuid.UUID `json:"id"`
	Amount    int32     `json:"amount"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

func toTransactionResponse(t Transaction) TransactionResponse {
	return TransactionResponse{ID: t.ID, Amount: t.Amount, Reason: t.Reason, CreatedAt: t.CreatedAt}
}
