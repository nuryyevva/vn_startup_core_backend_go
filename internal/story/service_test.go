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

func TestToSceneResponse_PreservesDialogueScript(t *testing.T) {
	raw := json.RawMessage(`[{"line":"Hello"}]`)
	scene := Scene{ID: uuid.New(), DialogueScript: raw}

	resp := toSceneResponse(scene, nil)

	assert.JSONEq(t, string(raw), string(resp.DialogueScript))
}
