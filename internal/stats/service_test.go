package stats

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/pkg/apperr"
)

var errBoom = errors.New("boom")

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetStats(ctx context.Context, userID uuid.UUID) (Stats, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(Stats), args.Error(1)
}

func (m *mockRepository) RecordHeartbeat(ctx context.Context, userID uuid.UUID, seconds int32, today time.Time) (Stats, error) {
	args := m.Called(ctx, userID, seconds, today)
	return args.Get(0).(Stats), args.Error(1)
}

func TestService_GetStats_ReturnsZeroWhenNeverHeartbeat(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("GetStats", mock.Anything, userID).Return(Stats{}, ErrStatsNotFound)

	stats, err := svc.GetStats(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, userID, stats.UserID)
	assert.Zero(t, stats.DayStreak)
}

func TestService_GetStats_PropagatesOtherErrors(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("GetStats", mock.Anything, userID).Return(Stats{}, errBoom)

	_, err := svc.GetStats(context.Background(), userID)

	require.ErrorIs(t, err, errBoom)
}

func TestService_Heartbeat_RejectsNonPositiveSeconds(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	_, err := svc.Heartbeat(context.Background(), uuid.New(), 0)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_amount", appErr.Code)
	repo.AssertNotCalled(t, "RecordHeartbeat", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Heartbeat_RejectsOversizedSeconds(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	_, err := svc.Heartbeat(context.Background(), uuid.New(), maxHeartbeatSeconds+1)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_amount", appErr.Code)
}

func TestService_Heartbeat_RecordsAgainstTruncatedUTCDate(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	fixedNow := time.Date(2026, 9, 23, 14, 30, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }

	userID := uuid.New()
	expectedDay := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	updated := Stats{UserID: userID, DayStreak: 2, TotalReadingSeconds: 90}
	repo.On("RecordHeartbeat", mock.Anything, userID, int32(30), expectedDay).Return(updated, nil)

	stats, err := svc.Heartbeat(context.Background(), userID, 30)

	require.NoError(t, err)
	assert.Equal(t, updated, stats)
}
