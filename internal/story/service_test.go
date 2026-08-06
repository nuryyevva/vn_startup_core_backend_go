package story

import (
	"context"
	"encoding/json"
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

func (m *mockRepository) ListPublishedStories(ctx context.Context) ([]Story, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Story), args.Error(1)
}

func (m *mockRepository) GetStory(ctx context.Context, id uuid.UUID) (Story, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Story), args.Error(1)
}

func (m *mockRepository) GetFirstScene(ctx context.Context, storyID uuid.UUID) (Scene, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).(Scene), args.Error(1)
}

func (m *mockRepository) GetScene(ctx context.Context, id uuid.UUID) (Scene, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Scene), args.Error(1)
}

func (m *mockRepository) ListChoices(ctx context.Context, sceneID uuid.UUID) ([]Choice, error) {
	args := m.Called(ctx, sceneID)
	return args.Get(0).([]Choice), args.Error(1)
}

func (m *mockRepository) GetChoice(ctx context.Context, id uuid.UUID) (Choice, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Choice), args.Error(1)
}

func (m *mockRepository) GetProgress(ctx context.Context, userID, storyID uuid.UUID) (Progress, error) {
	args := m.Called(ctx, userID, storyID)
	return args.Get(0).(Progress), args.Error(1)
}

func (m *mockRepository) CreateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID) (Progress, error) {
	args := m.Called(ctx, userID, storyID, sceneID)
	return args.Get(0).(Progress), args.Error(1)
}

func (m *mockRepository) UpdateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID, choicesMade []uuid.UUID) (Progress, error) {
	args := m.Called(ctx, userID, storyID, sceneID, choicesMade)
	return args.Get(0).(Progress), args.Error(1)
}

func (m *mockRepository) ListProgressByUser(ctx context.Context, userID uuid.UUID) ([]ProgressSummary, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]ProgressSummary), args.Error(1)
}

func (m *mockRepository) ListScenesByStory(ctx context.Context, storyID uuid.UUID) ([]Scene, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).([]Scene), args.Error(1)
}

func (m *mockRepository) IsSceneUnlocked(ctx context.Context, userID, sceneID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, sceneID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepository) ListUnlockedSceneIDs(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}

func (m *mockRepository) UnlockScene(ctx context.Context, userID, sceneID uuid.UUID) error {
	args := m.Called(ctx, userID, sceneID)
	return args.Error(0)
}

type mockWallet struct {
	mock.Mock
}

func (m *mockWallet) Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) error {
	args := m.Called(ctx, userID, amount, reason)
	return args.Error(0)
}

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload any) error {
	args := m.Called(ctx, subject, payload)
	return args.Error(0)
}

func TestService_GetOrCreateProgress_CreatesWhenMissing(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID, storyID, firstSceneID := uuid.New(), uuid.New(), uuid.New()

	repo.On("GetProgress", mock.Anything, userID, storyID).Return(Progress{}, ErrProgressNotFound)
	repo.On("GetStory", mock.Anything, storyID).Return(Story{ID: storyID, IsPublished: true}, nil)
	repo.On("GetFirstScene", mock.Anything, storyID).Return(Scene{ID: firstSceneID, StoryID: storyID}, nil)
	created := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: firstSceneID}
	repo.On("CreateProgress", mock.Anything, userID, storyID, firstSceneID).Return(created, nil)

	progress, err := svc.GetOrCreateProgress(context.Background(), userID, storyID)

	require.NoError(t, err)
	assert.Equal(t, created, progress)
}

func TestService_GetOrCreateProgress_ReturnsExisting(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID, storyID := uuid.New(), uuid.New()
	existing := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: uuid.New()}
	repo.On("GetProgress", mock.Anything, userID, storyID).Return(existing, nil)

	progress, err := svc.GetOrCreateProgress(context.Background(), userID, storyID)

	require.NoError(t, err)
	assert.Equal(t, existing, progress)
	repo.AssertNotCalled(t, "CreateProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestService_SubmitChoice_FreeChoiceAdvancesScene(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	publisher := new(mockPublisher)
	svc := NewService(repo, wallet, publisher, nil)

	userID, storyID := uuid.New(), uuid.New()
	sceneID, nextSceneID, choiceID := uuid.New(), uuid.New(), uuid.New()

	scene := Scene{ID: sceneID, StoryID: storyID}
	choice := Choice{ID: choiceID, SceneID: sceneID, IsPaid: false, NextSceneID: &nextSceneID}
	existingProgress := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: sceneID, ChoicesMade: nil}
	updated := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: nextSceneID, ChoicesMade: []uuid.UUID{choiceID}}

	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("GetChoice", mock.Anything, choiceID).Return(choice, nil)
	repo.On("GetProgress", mock.Anything, userID, storyID).Return(existingProgress, nil)
	repo.On("UpdateProgress", mock.Anything, userID, storyID, nextSceneID, []uuid.UUID{choiceID}).Return(updated, nil)
	publisher.On("Publish", mock.Anything, "player.choice.made", mock.Anything).Return(nil)

	progress, err := svc.SubmitChoice(context.Background(), userID, sceneID, choiceID)

	require.NoError(t, err)
	assert.Equal(t, updated, progress)
	wallet.AssertNotCalled(t, "Debit", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	publisher.AssertExpectations(t)
}

func TestService_SubmitChoice_PaidChoiceDebitsWallet(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	publisher := new(mockPublisher)
	svc := NewService(repo, wallet, publisher, nil)

	userID, storyID := uuid.New(), uuid.New()
	sceneID, nextSceneID, choiceID := uuid.New(), uuid.New(), uuid.New()

	scene := Scene{ID: sceneID, StoryID: storyID}
	choice := Choice{ID: choiceID, SceneID: sceneID, IsPaid: true, CostDiamonds: 50, NextSceneID: &nextSceneID}
	existingProgress := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: sceneID}
	updated := Progress{UserID: userID, StoryID: storyID, CurrentSceneID: nextSceneID, ChoicesMade: []uuid.UUID{choiceID}}

	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("GetChoice", mock.Anything, choiceID).Return(choice, nil)
	wallet.On("Debit", mock.Anything, userID, int32(50), "choice_purchase").Return(nil)
	repo.On("GetProgress", mock.Anything, userID, storyID).Return(existingProgress, nil)
	repo.On("UpdateProgress", mock.Anything, userID, storyID, nextSceneID, []uuid.UUID{choiceID}).Return(updated, nil)
	publisher.On("Publish", mock.Anything, "player.choice.made", mock.Anything).Return(nil)

	progress, err := svc.SubmitChoice(context.Background(), userID, sceneID, choiceID)

	require.NoError(t, err)
	assert.Equal(t, updated, progress)
	wallet.AssertExpectations(t)
}

func TestService_SubmitChoice_InsufficientFundsStopsProgress(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	svc := NewService(repo, wallet, new(mockPublisher), nil)

	userID, storyID := uuid.New(), uuid.New()
	sceneID, choiceID := uuid.New(), uuid.New()

	scene := Scene{ID: sceneID, StoryID: storyID}
	choice := Choice{ID: choiceID, SceneID: sceneID, IsPaid: true, CostDiamonds: 999}

	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("GetChoice", mock.Anything, choiceID).Return(choice, nil)
	insufficientErr := apperr.New(402, "insufficient_funds", "not enough diamonds")
	wallet.On("Debit", mock.Anything, userID, int32(999), "choice_purchase").Return(insufficientErr)

	_, err := svc.SubmitChoice(context.Background(), userID, sceneID, choiceID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "insufficient_funds", appErr.Code)
	repo.AssertNotCalled(t, "UpdateProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestService_SubmitChoice_ChoiceNotInScene(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	sceneID, otherSceneID, choiceID := uuid.New(), uuid.New(), uuid.New()
	repo.On("GetScene", mock.Anything, sceneID).Return(Scene{ID: sceneID}, nil)
	repo.On("GetChoice", mock.Anything, choiceID).Return(Choice{ID: choiceID, SceneID: otherSceneID}, nil)

	_, err := svc.SubmitChoice(context.Background(), uuid.New(), sceneID, choiceID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "choice_not_in_scene", appErr.Code)
}

func TestService_GetSceneForPlayer_FreeSceneNeedsNoUnlock(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	sceneID := uuid.New()
	scene := Scene{ID: sceneID}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("ListChoices", mock.Anything, sceneID).Return([]Choice{}, nil)

	got, _, err := svc.GetSceneForPlayer(context.Background(), uuid.New(), sceneID)

	require.NoError(t, err)
	assert.Equal(t, scene, got)
	repo.AssertNotCalled(t, "IsSceneUnlocked", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_GetSceneForPlayer_LockedSceneDeniesAccess(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID, sceneID := uuid.New(), uuid.New()
	cost := int32(25)
	scene := Scene{ID: sceneID, UnlockCostDiamonds: &cost}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("IsSceneUnlocked", mock.Anything, userID, sceneID).Return(false, nil)

	_, _, err := svc.GetSceneForPlayer(context.Background(), userID, sceneID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "scene_locked", appErr.Code)
	repo.AssertNotCalled(t, "ListChoices", mock.Anything, mock.Anything)
}

func TestService_GetSceneForPlayer_UnlockedPaidSceneGrantsAccess(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID, sceneID := uuid.New(), uuid.New()
	cost := int32(25)
	scene := Scene{ID: sceneID, UnlockCostDiamonds: &cost}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("IsSceneUnlocked", mock.Anything, userID, sceneID).Return(true, nil)
	repo.On("ListChoices", mock.Anything, sceneID).Return([]Choice{}, nil)

	got, _, err := svc.GetSceneForPlayer(context.Background(), userID, sceneID)

	require.NoError(t, err)
	assert.Equal(t, scene, got)
}

func TestService_UnlockScene_DebitsWalletAndRecordsUnlock(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	svc := NewService(repo, wallet, new(mockPublisher), nil)

	userID, sceneID := uuid.New(), uuid.New()
	cost := int32(25)
	scene := Scene{ID: sceneID, UnlockCostDiamonds: &cost}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("IsSceneUnlocked", mock.Anything, userID, sceneID).Return(false, nil)
	wallet.On("Debit", mock.Anything, userID, int32(25), "scene_unlock").Return(nil)
	repo.On("UnlockScene", mock.Anything, userID, sceneID).Return(nil)

	err := svc.UnlockScene(context.Background(), userID, sceneID)

	require.NoError(t, err)
	wallet.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestService_UnlockScene_AlreadyUnlockedIsIdempotent(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	svc := NewService(repo, wallet, new(mockPublisher), nil)

	userID, sceneID := uuid.New(), uuid.New()
	cost := int32(25)
	scene := Scene{ID: sceneID, UnlockCostDiamonds: &cost}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("IsSceneUnlocked", mock.Anything, userID, sceneID).Return(true, nil)

	err := svc.UnlockScene(context.Background(), userID, sceneID)

	require.NoError(t, err)
	wallet.AssertNotCalled(t, "Debit", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "UnlockScene", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_UnlockScene_SceneWithoutCostRejected(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	sceneID := uuid.New()
	repo.On("GetScene", mock.Anything, sceneID).Return(Scene{ID: sceneID}, nil)

	err := svc.UnlockScene(context.Background(), uuid.New(), sceneID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "scene_not_locked", appErr.Code)
}

func TestService_UnlockScene_InsufficientFundsPropagates(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	svc := NewService(repo, wallet, new(mockPublisher), nil)

	userID, sceneID := uuid.New(), uuid.New()
	cost := int32(999)
	scene := Scene{ID: sceneID, UnlockCostDiamonds: &cost}
	repo.On("GetScene", mock.Anything, sceneID).Return(scene, nil)
	repo.On("IsSceneUnlocked", mock.Anything, userID, sceneID).Return(false, nil)
	insufficientErr := apperr.New(402, "insufficient_funds", "not enough diamonds")
	wallet.On("Debit", mock.Anything, userID, int32(999), "scene_unlock").Return(insufficientErr)

	err := svc.UnlockScene(context.Background(), userID, sceneID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "insufficient_funds", appErr.Code)
	repo.AssertNotCalled(t, "UnlockScene", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ListStoryScenes_MarksLockStateFromUnlockedSet(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID, storyID := uuid.New(), uuid.New()
	freeSceneID, paidLockedID, paidUnlockedID := uuid.New(), uuid.New(), uuid.New()
	cost := int32(25)

	repo.On("GetStory", mock.Anything, storyID).Return(Story{ID: storyID}, nil)
	repo.On("ListScenesByStory", mock.Anything, storyID).Return([]Scene{
		{ID: freeSceneID, OrderIndex: 0},
		{ID: paidLockedID, OrderIndex: 1, UnlockCostDiamonds: &cost},
		{ID: paidUnlockedID, OrderIndex: 2, UnlockCostDiamonds: &cost},
	}, nil)
	repo.On("ListUnlockedSceneIDs", mock.Anything, userID).Return(map[uuid.UUID]bool{paidUnlockedID: true}, nil)

	scenes, unlocked, err := svc.ListStoryScenes(context.Background(), userID, storyID)

	require.NoError(t, err)
	require.Len(t, scenes, 3)
	responses := toSceneSummaryResponses(scenes, unlocked)
	assert.True(t, responses[0].IsUnlocked)
	assert.False(t, responses[1].IsUnlocked)
	assert.True(t, responses[2].IsUnlocked)
}

func TestService_ListMyProgress_MarksFinishedFromOrderIndex(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, new(mockWallet), new(mockPublisher), nil)

	userID := uuid.New()
	summaries := []ProgressSummary{
		{Story: Story{ID: uuid.New(), Title: "In progress"}, CurrentOrderIndex: 1, TotalScenes: 5},
		{Story: Story{ID: uuid.New(), Title: "Finished"}, CurrentOrderIndex: 4, TotalScenes: 5},
	}
	repo.On("ListProgressByUser", mock.Anything, userID).Return(summaries, nil)

	got, err := svc.ListMyProgress(context.Background(), userID)

	require.NoError(t, err)
	require.Len(t, got, 2)
	responses := toProgressSummaryResponses(got)
	assert.False(t, responses[0].IsFinished)
	assert.True(t, responses[1].IsFinished)
}

func TestToSceneResponse_PreservesDialogueScript(t *testing.T) {
	raw := json.RawMessage(`[{"line":"Hello"}]`)
	scene := Scene{ID: uuid.New(), DialogueScript: raw}

	resp := toSceneResponse(scene, nil)

	assert.JSONEq(t, string(raw), string(resp.DialogueScript))
}
