package wallet

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/pkg/apperr"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockRepository) Credit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error) {
	args := m.Called(ctx, userID, amount, reason)
	return args.Get(0).(Transaction), args.Error(1)
}

func (m *mockRepository) Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) (Transaction, error) {
	args := m.Called(ctx, userID, amount, reason)
	return args.Get(0).(Transaction), args.Error(1)
}

func (m *mockRepository) GetTotalSpent(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func TestService_Balance(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("GetBalance", mock.Anything, userID).Return(int64(120), nil)

	balance, err := svc.Balance(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, int64(120), balance)
}

func TestService_Grant_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	tx := Transaction{ID: uuid.New(), UserID: userID, Amount: 500, Reason: ReasonDevGrant}
	repo.On("Credit", mock.Anything, userID, int32(500), ReasonDevGrant).Return(tx, nil)

	got, err := svc.Grant(context.Background(), userID, 500)

	require.NoError(t, err)
	assert.Equal(t, tx, got)
}

func TestService_Grant_RejectsNonPositiveAmount(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	_, err := svc.Grant(context.Background(), uuid.New(), 0)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_amount", appErr.Code)
	repo.AssertNotCalled(t, "Credit")
}

func TestService_Debit_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("Debit", mock.Anything, userID, int32(20), "dialog_message").Return(Transaction{}, nil)

	err := svc.Debit(context.Background(), userID, 20, "dialog_message")

	require.NoError(t, err)
}

func TestService_Debit_InsufficientFunds(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("Debit", mock.Anything, userID, int32(9999), "choice_purchase").Return(Transaction{}, ErrInsufficientFunds)

	err := svc.Debit(context.Background(), userID, 9999, "choice_purchase")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "insufficient_funds", appErr.Code)
	assert.Equal(t, 402, appErr.Status)
}

func TestService_Debit_RejectsNonPositiveAmount(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	err := svc.Debit(context.Background(), uuid.New(), -5, "choice_purchase")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_amount", appErr.Code)
	repo.AssertNotCalled(t, "Debit")
}
