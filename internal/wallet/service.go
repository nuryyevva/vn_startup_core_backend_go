package wallet

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Balance(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.GetBalance(ctx, userID)
}

// TotalSpent returns the sum of all diamonds ever debited from userID, as a
// positive number.
func (s *Service) TotalSpent(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.GetTotalSpent(ctx, userID)
}

// Grant credits amount diamonds to userID. It is only ever called from the
// dev-only /dev/wallet/grant endpoint, gated by Config.Dev.EnableDevEndpoints.
func (s *Service) Grant(ctx context.Context, userID uuid.UUID, amount int32) (Transaction, error) {
	if amount <= 0 {
		return Transaction{}, apperr.BadRequest("invalid_amount", "Сумма начисления должна быть положительной")
	}
	return s.repo.Credit(ctx, userID, amount, ReasonDevGrant)
}

// Debit satisfies the Wallet interface expected by the story and dialog
// modules: charge amount diamonds to userID for reason, or fail with a
// well-formed insufficient_funds error.
func (s *Service) Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) error {
	if amount <= 0 {
		return apperr.BadRequest("invalid_amount", "Сумма списания должна быть положительной")
	}

	_, err := s.repo.Debit(ctx, userID, amount, reason)
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return apperr.New(http.StatusPaymentRequired, "insufficient_funds", "Недостаточно алмазов для этого действия")
		}
		return err
	}
	return nil
}
