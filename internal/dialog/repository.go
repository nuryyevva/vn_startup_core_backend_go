package dialog

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound  = errors.New("dialog session not found")
	ErrSessionNotActive = errors.New("dialog session is not active")
)

type Repository interface {
	CreateSession(ctx context.Context, userID, characterID, sceneID uuid.UUID, limitType string, limitValue *int32) (Session, error)
	GetSession(ctx context.Context, id uuid.UUID) (Session, error)
	IncrementMessageCount(ctx context.Context, id uuid.UUID) (Session, error)
	EndSession(ctx context.Context, id uuid.UUID, reason string) (Session, error)
	ListExpiredTimeLimitedSessions(ctx context.Context) ([]Session, error)
	CreateMessage(ctx context.Context, sessionID uuid.UUID, sender, text string, cost int32) (Message, error)
	ListMessages(ctx context.Context, sessionID uuid.UUID) ([]Message, error)
	// CountUserMessages returns how many messages with the given sender
	// (SenderUser or SenderCharacter) userID has ever sent across all of
	// their dialog sessions — used by the achievement module.
	CountUserMessages(ctx context.Context, userID uuid.UUID, sender string) (int64, error)
}
