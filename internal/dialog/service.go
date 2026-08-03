package dialog

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/events"
)

// SceneProvider is the subset of the story module needed to validate that a
// scene allows free dialog and to read its default limit configuration.
type SceneProvider interface {
	GetSceneInfo(ctx context.Context, sceneID uuid.UUID) (SceneInfo, error)
}

type SceneInfo struct {
	ID                uuid.UUID
	FreeDialogEnabled bool
	DialogLimitType   *string
	DialogLimitValue  *int32
}

// Wallet is the subset of the wallet module's service the dialog module
// needs to charge for free-dialog messages.
type Wallet interface {
	Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) error
}

type Service struct {
	repo               Repository
	scenes             SceneProvider
	wallet             Wallet
	publisher          EventPublisher
	notifier           Notifier
	limiter            *SessionLimiter
	defaultMessageCost int32
	logger             *slog.Logger
}

func NewService(
	repo Repository,
	scenes SceneProvider,
	wallet Wallet,
	publisher EventPublisher,
	notifier Notifier,
	defaultMessageCost int32,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:               repo,
		scenes:             scenes,
		wallet:             wallet,
		publisher:          publisher,
		notifier:           notifier,
		limiter:            NewSessionLimiter(),
		defaultMessageCost: defaultMessageCost,
		logger:             logger,
	}
}

func (s *Service) StartSession(ctx context.Context, userID, characterID, sceneID uuid.UUID) (Session, error) {
	scene, err := s.scenes.GetSceneInfo(ctx, sceneID)
	if err != nil {
		return Session{}, err
	}
	if !scene.FreeDialogEnabled {
		return Session{}, apperr.Forbidden("free_dialog_disabled", "Свободный диалог недоступен для этой сцены")
	}

	limitType := LimitTypeNone
	if scene.DialogLimitType != nil {
		limitType = *scene.DialogLimitType
	}

	session, err := s.repo.CreateSession(ctx, userID, characterID, sceneID, limitType, scene.DialogLimitValue)
	if err != nil {
		return Session{}, err
	}

	s.publish(ctx, events.SubjectDialogSessionStart, SessionStartedEvent{
		SessionID:   session.ID,
		UserID:      session.UserID,
		CharacterID: session.CharacterID,
		SceneID:     session.SceneID,
	})

	return session, nil
}

func (s *Service) GetSession(ctx context.Context, userID, sessionID uuid.UUID) (Session, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Session{}, apperr.NotFound("session_not_found", "Сессия диалога не найдена")
		}
		return Session{}, err
	}
	if session.UserID != userID {
		return Session{}, apperr.NotFound("session_not_found", "Сессия диалога не найдена")
	}
	return session, nil
}

// SessionOwner returns the user ID that owns sessionID, regardless of who
// is asking. Used by the realtime subscriber to route AI events (which
// carry only a session_id) to the right WebSocket connection.
func (s *Service) SessionOwner(ctx context.Context, sessionID uuid.UUID) (uuid.UUID, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return uuid.Nil, err
	}
	return session.UserID, nil
}

// SendMessage records the user's message, charges diamonds for it, and
// ends the session if this message exhausted its limit.
func (s *Service) SendMessage(ctx context.Context, userID, sessionID uuid.UUID, text string) (Message, *int32, error) {
	session, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return Message{}, nil, err
	}
	if session.Status != StatusActive {
		return Message{}, nil, apperr.Conflict("session_not_active", "Эта сессия диалога уже завершена")
	}

	if allowed, reason := s.limiter.CheckLimit(session); !allowed {
		s.endSessionInternal(ctx, session, reason)
		return Message{}, nil, apperr.Conflict("session_ended", "Лимит сессии диалога исчерпан")
	}

	if err := s.wallet.Debit(ctx, userID, s.defaultMessageCost, "dialog_message"); err != nil {
		return Message{}, nil, err
	}

	session, err = s.repo.IncrementMessageCount(ctx, sessionID)
	if err != nil {
		return Message{}, nil, err
	}

	message, err := s.repo.CreateMessage(ctx, sessionID, SenderUser, text, s.defaultMessageCost)
	if err != nil {
		return Message{}, nil, err
	}

	remaining := s.limiter.RemainingLimit(session)

	s.publish(ctx, events.SubjectDialogMessageSent, MessageSentEvent{
		SessionID:      session.ID,
		MessageID:      message.ID,
		UserID:         session.UserID,
		CharacterID:    session.CharacterID,
		Text:           message.Text,
		RemainingLimit: remaining,
	})

	if allowed, reason := s.limiter.CheckLimit(session); !allowed {
		s.endSessionInternal(ctx, session, reason)
	}

	return message, remaining, nil
}

// RecordCharacterMessage persists an AI-generated reply once the AI
// Orchestrator publishes ai.dialog.response.complete. Called from the
// realtime NATS subscriber, never directly from an HTTP handler.
func (s *Service) RecordCharacterMessage(ctx context.Context, sessionID uuid.UUID, text string) (Message, error) {
	return s.repo.CreateMessage(ctx, sessionID, SenderCharacter, text, 0)
}

// EndSession is called when the player deliberately exits a session
// (POST /dialog/sessions/:id/end).
func (s *Service) EndSession(ctx context.Context, userID, sessionID uuid.UUID) (Session, error) {
	session, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusActive {
		return Session{}, apperr.Conflict("session_not_active", "Эта сессия диалога уже завершена")
	}

	return s.endSessionInternal(ctx, session, EndReasonUser), nil
}

// endSessionInternal ends a session for any reason (user request, message
// limit, time limit) and fans the result out over NATS and the WebSocket.
func (s *Service) endSessionInternal(ctx context.Context, session Session, reason string) Session {
	ended, err := s.repo.EndSession(ctx, session.ID, reason)
	if err != nil {
		if errors.Is(err, ErrSessionNotActive) || errors.Is(err, ErrSessionNotFound) {
			// Already ended by a concurrent request/scheduler tick.
			return session
		}
		s.logger.Error("end dialog session failed", slog.String("session_id", session.ID.String()), slog.String("error", err.Error()))
		return session
	}

	s.publish(ctx, events.SubjectDialogSessionEnded, SessionEndedEvent{SessionID: ended.ID, Reason: reason})

	if s.notifier != nil {
		payload, err := json.Marshal(wsSessionEndedMessage{Type: "session_ended", SessionID: ended.ID, Reason: reason})
		if err != nil {
			s.logger.Error("marshal session_ended ws message failed", slog.String("error", err.Error()))
		} else if err := s.notifier.SendToUser(ended.UserID.String(), payload); err != nil {
			s.logger.Warn("send session_ended ws message failed", slog.String("user_id", ended.UserID.String()), slog.String("error", err.Error()))
		}
	}

	return ended
}

// EndExpiredSessions is invoked periodically by the scheduler to close
// active time-limited sessions whose deadline has passed.
func (s *Service) EndExpiredSessions(ctx context.Context) (int, error) {
	expired, err := s.repo.ListExpiredTimeLimitedSessions(ctx)
	if err != nil {
		return 0, err
	}
	for _, session := range expired {
		s.endSessionInternal(ctx, session, EndReasonTimeLimit)
	}
	return len(expired), nil
}

func (s *Service) publish(ctx context.Context, subject string, payload any) {
	if s.publisher == nil {
		return
	}
	if err := s.publisher.Publish(ctx, subject, payload); err != nil {
		s.logger.Error("publish event failed", slog.String("subject", subject), slog.String("error", err.Error()))
	}
}
