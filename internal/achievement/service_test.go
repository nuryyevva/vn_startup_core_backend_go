package achievement

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) ListUnlocked(ctx context.Context, userID uuid.UUID) (map[string]time.Time, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]time.Time), args.Error(1)
}

func (m *mockRepository) Unlock(ctx context.Context, userID uuid.UUID, achievementID string) (time.Time, error) {
	args := m.Called(ctx, userID, achievementID)
	return args.Get(0).(time.Time), args.Error(1)
}

type mockActivityProvider struct {
	mock.Mock
}

func (m *mockActivityProvider) GetActivityStats(ctx context.Context, userID uuid.UUID) (ActivityStats, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(ActivityStats), args.Error(1)
}

func TestService_ListAchievements_AllLockedWhenNoActivity(t *testing.T) {
	repo := new(mockRepository)
	activity := new(mockActivityProvider)
	svc := NewService(repo, activity)

	userID := uuid.New()
	repo.On("ListUnlocked", mock.Anything, userID).Return(map[string]time.Time{}, nil)
	activity.On("GetActivityStats", mock.Anything, userID).Return(ActivityStats{}, nil)

	achievements, err := svc.ListAchievements(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, achievements, len(Catalog))
	for _, a := range achievements {
		assert.False(t, a.Unlocked)
		assert.Nil(t, a.UnlockedAt)
	}
	repo.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ListAchievements_PreservesAlreadyUnlockedTimestamp(t *testing.T) {
	repo := new(mockRepository)
	activity := new(mockActivityProvider)
	svc := NewService(repo, activity)

	userID := uuid.New()
	unlockedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.On("ListUnlocked", mock.Anything, userID).Return(map[string]time.Time{"first_story_finished": unlockedAt}, nil)
	activity.On("GetActivityStats", mock.Anything, userID).Return(ActivityStats{}, nil)

	achievements, err := svc.ListAchievements(context.Background(), userID)

	require.NoError(t, err)
	found := findAchievement(achievements, "first_story_finished")
	require.NotNil(t, found)
	assert.True(t, found.Unlocked)
	require.NotNil(t, found.UnlockedAt)
	assert.Equal(t, unlockedAt, *found.UnlockedAt)
	repo.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, "first_story_finished")
}

func TestService_ListAchievements_UnlocksNewlyQualifyingAchievement(t *testing.T) {
	repo := new(mockRepository)
	activity := new(mockActivityProvider)
	svc := NewService(repo, activity)

	userID := uuid.New()
	now := time.Now().UTC()
	repo.On("ListUnlocked", mock.Anything, userID).Return(map[string]time.Time{}, nil)
	activity.On("GetActivityStats", mock.Anything, userID).Return(ActivityStats{FinishedStories: 1}, nil)
	repo.On("Unlock", mock.Anything, userID, "first_story_finished").Return(now, nil)

	achievements, err := svc.ListAchievements(context.Background(), userID)

	require.NoError(t, err)
	found := findAchievement(achievements, "first_story_finished")
	require.NotNil(t, found)
	assert.True(t, found.Unlocked)
	require.NotNil(t, found.UnlockedAt)
	assert.Equal(t, now, *found.UnlockedAt)
}

func findAchievement(achievements []Achievement, id string) *Achievement {
	for i := range achievements {
		if achievements[i].ID == id {
			return &achievements[i]
		}
	}
	return nil
}
