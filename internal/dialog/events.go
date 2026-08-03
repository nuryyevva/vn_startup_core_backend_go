package dialog

import (
	"context"

	"github.com/google/uuid"
)

// EventPublisher is the subset of pkg/events.Publisher the dialog module
// needs; it's an interface here so unit tests can supply a fake.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// Notifier delivers a JSON payload to every WebSocket connection belonging
// to userID. It matches realtime.Hub.SendToUser structurally so the dialog
// module never needs to import the realtime package.
type Notifier interface {
	SendToUser(userID string, payload []byte) error
}

type SessionStartedEvent struct {
	SessionID   uuid.UUID `json:"session_id"`
	UserID      uuid.UUID `json:"user_id"`
	CharacterID uuid.UUID `json:"character_id"`
	SceneID     uuid.UUID `json:"scene_id"`
}

type MessageSentEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	MessageID      uuid.UUID `json:"message_id"`
	UserID         uuid.UUID `json:"user_id"`
	CharacterID    uuid.UUID `json:"character_id"`
	Text           string    `json:"text"`
	RemainingLimit *int32    `json:"remaining_limit"`
}

type SessionEndedEvent struct {
	SessionID uuid.UUID `json:"session_id"`
	Reason    string    `json:"reason"`
}

// wsSessionEndedMessage is the WebSocket payload pushed to the session's
// owner when it ends, matching the protocol documented for /ws:
// {"type": "session_ended", "session_id": "...", "reason": "..."}.
type wsSessionEndedMessage struct {
	Type      string    `json:"type"`
	SessionID uuid.UUID `json:"session_id"`
	Reason    string    `json:"reason"`
}
