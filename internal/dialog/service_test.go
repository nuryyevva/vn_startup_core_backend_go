package dialog

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

func (m *mockRepository) CreateSession(ctx context.Context, userID, characterID, sceneID uuid.UUID, limitType string, limitValue *int32) (Session, error) {
	args := m.Called(ctx, userID, characterID, sceneID, limitType, limitValue)
	return args.Get(0).(Session), args.Error(1)
}

func (m *mockRepository) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Session), args.Error(1)
}

func (m *mockRepository) IncrementMessageCount(ctx context.Context, id uuid.UUID) (Session, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Session), args.Error(1)
}

func (m *mockRepository) EndSession(ctx context.Context, id uuid.UUID, reason string) (Session, error) {
	args := m.Called(ctx, id, reason)
	return args.Get(0).(Session), args.Error(1)
}

func (m *mockRepository) ListExpiredTimeLimitedSessions(ctx context.Context) ([]Session, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Session), args.Error(1)
}

func (m *mockRepository) CreateMessage(ctx context.Context, sessionID uuid.UUID, sender, text string, cost int32) (Message, error) {
	args := m.Called(ctx, sessionID, sender, text, cost)
	return args.Get(0).(Message), args.Error(1)
}

func (m *mockRepository) ListMessages(ctx context.Context, sessionID uuid.UUID) ([]Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *mockRepository) CountUserMessages(ctx context.Context, userID uuid.UUID, sender string) (int64, error) {
	args := m.Called(ctx, userID, sender)
	return args.Get(0).(int64), args.Error(1)
}

type mockSceneProvider struct {
	mock.Mock
}

func (m *mockSceneProvider) GetSceneInfo(ctx context.Context, sceneID uuid.UUID) (SceneInfo, error) {
	args := m.Called(ctx, sceneID)
	return args.Get(0).(SceneInfo), args.Error(1)
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

type mockNotifier struct {
	mock.Mock
}

func (m *mockNotifier) SendToUser(userID string, payload []byte) error {
	args := m.Called(userID, payload)
	return args.Error(0)
}

func newTestService(repo *mockRepository, scenes *mockSceneProvider, wallet *mockWallet, publisher *mockPublisher, notifier *mockNotifier) *Service {
	return NewService(repo, scenes, wallet, publisher, notifier, 20, nil)
}

func TestService_StartSession_Success(t *testing.T) {
	repo := new(mockRepository)
	scenes := new(mockSceneProvider)
	publisher := new(mockPublisher)
	svc := newTestService(repo, scenes, new(mockWallet), publisher, new(mockNotifier))

	userID, characterID, sceneID := uuid.New(), uuid.New(), uuid.New()
	limitValue := int32(5)
	scenes.On("GetSceneInfo", mock.Anything, sceneID).Return(SceneInfo{
		ID: sceneID, FreeDialogEnabled: true, DialogLimitType: strPtr(LimitTypeMessages), DialogLimitValue: &limitValue,
	}, nil)

	created := Session{ID: uuid.New(), UserID: userID, CharacterID: characterID, SceneID: sceneID, Status: StatusActive}
	repo.On("CreateSession", mock.Anything, userID, characterID, sceneID, LimitTypeMessages, &limitValue).Return(created, nil)
	publisher.On("Publish", mock.Anything, "dialog.session.started", mock.Anything).Return(nil)

	session, err := svc.StartSession(context.Background(), userID, characterID, sceneID)

	require.NoError(t, err)
	assert.Equal(t, created, session)
}

func TestService_StartSession_FreeDialogDisabled(t *testing.T) {
	repo := new(mockRepository)
	scenes := new(mockSceneProvider)
	svc := newTestService(repo, scenes, new(mockWallet), new(mockPublisher), new(mockNotifier))

	sceneID := uuid.New()
	scenes.On("GetSceneInfo", mock.Anything, sceneID).Return(SceneInfo{ID: sceneID, FreeDialogEnabled: false}, nil)

	_, err := svc.StartSession(context.Background(), uuid.New(), uuid.New(), sceneID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "free_dialog_disabled", appErr.Code)
	repo.AssertNotCalled(t, "CreateSession")
}

func TestService_SendMessage_Success(t *testing.T) {
	repo := new(mockRepository)
	wallet := new(mockWallet)
	publisher := new(mockPublisher)
	svc := newTestService(repo, new(mockSceneProvider), wallet, publisher, new(mockNotifier))

	userID, sessionID := uuid.New(), uuid.New()
	session := Session{ID: sessionID, UserID: userID, Status: StatusActive, LimitType: LimitTypeNone}

	repo.On("GetSession", mock.Anything, sessionID).Return(session, nil)
	wallet.On("Debit", mock.Anything, userID, int32(20), "dialog_message").Return(nil)
	incremented := session
	incremented.MessageCount = 1
	repo.On("IncrementMessageCount", mock.Anything, sessionID).Return(incremented, nil)
	msg := Message{ID: uuid.New(), SessionID: sessionID, Sender: SenderUser, Text: "hello", CostDiamonds: 20}
	repo.On("CreateMessage", mock.Anything, sessionID, SenderUser, "hello", int32(20)).Return(msg, nil)
	publisher.On("Publish", mock.Anything, "dialog.message.sent", mock.Anything).Return(nil)

	got, remaining, err := svc.SendMessage(context.Background(), userID, sessionID, "hello")

	require.NoError(t, err)
	assert.Equal(t, msg, got)
	assert.Nil(t, remaining)
}

func TestService_SendMessage_WrongOwnerNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockSceneProvider), new(mockWallet), new(mockPublisher), new(mockNotifier))

	sessionID := uuid.New()
	session := Session{ID: sessionID, UserID: uuid.New(), Status: StatusActive}
	repo.On("GetSession", mock.Anything, sessionID).Return(session, nil)

	_, _, err := svc.SendMessage(context.Background(), uuid.New(), sessionID, "hello")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "session_not_found", appErr.Code)
}

func TestService_SendMessage_LimitExceededEndsSession(t *testing.T) {
	repo := new(mockRepository)
	publisher := new(mockPublisher)
	notifier := new(mockNotifier)
	svc := newTestService(repo, new(mockSceneProvider), new(mockWallet), publisher, notifier)

	userID, sessionID := uuid.New(), uuid.New()
	limitValue := int32(3)
	session := Session{ID: sessionID, UserID: userID, Status: StatusActive, LimitType: LimitTypeMessages, LimitValue: &limitValue, MessageCount: 3}

	repo.On("GetSession", mock.Anything, sessionID).Return(session, nil)
	ended := session
	ended.Status = StatusEnded
	repo.On("EndSession", mock.Anything, sessionID, EndReasonMessageLimit).Return(ended, nil)
	publisher.On("Publish", mock.Anything, "dialog.session.ended", mock.Anything).Return(nil)
	notifier.On("SendToUser", userID.String(), mock.Anything).Return(nil)

	_, _, err := svc.SendMessage(context.Background(), userID, sessionID, "hello")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "session_ended", appErr.Code)
	repo.AssertCalled(t, "EndSession", mock.Anything, sessionID, EndReasonMessageLimit)
}

func TestService_EndSession_Success(t *testing.T) {
	repo := new(mockRepository)
	publisher := new(mockPublisher)
	notifier := new(mockNotifier)
	svc := newTestService(repo, new(mockSceneProvider), new(mockWallet), publisher, notifier)

	userID, sessionID := uuid.New(), uuid.New()
	session := Session{ID: sessionID, UserID: userID, Status: StatusActive}
	ended := session
	ended.Status = StatusEnded

	repo.On("GetSession", mock.Anything, sessionID).Return(session, nil)
	repo.On("EndSession", mock.Anything, sessionID, EndReasonUser).Return(ended, nil)
	publisher.On("Publish", mock.Anything, "dialog.session.ended", mock.Anything).Return(nil)
	notifier.On("SendToUser", userID.String(), mock.Anything).Return(nil)

	got, err := svc.EndSession(context.Background(), userID, sessionID)

	require.NoError(t, err)
	assert.Equal(t, StatusEnded, got.Status)
}

func TestService_EndSession_AlreadyEnded(t *testing.T) {
	repo := new(mockRepository)
	svc := newTestService(repo, new(mockSceneProvider), new(mockWallet), new(mockPublisher), new(mockNotifier))

	userID, sessionID := uuid.New(), uuid.New()
	session := Session{ID: sessionID, UserID: userID, Status: StatusEnded}
	repo.On("GetSession", mock.Anything, sessionID).Return(session, nil)

	_, err := svc.EndSession(context.Background(), userID, sessionID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "session_not_active", appErr.Code)
	repo.AssertNotCalled(t, "EndSession")
}

func strPtr(s string) *string { return &s }
