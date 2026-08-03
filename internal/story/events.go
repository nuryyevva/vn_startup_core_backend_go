package story

import (
	"context"

	"github.com/google/uuid"
)

// EventPublisher is the subset of pkg/events.Publisher the story module
// needs; it's an interface here so unit tests can supply a fake.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// PlayerChoiceMadeEvent is published to NATS subject "player.choice.made"
// whenever a player submits a choice.
type PlayerChoiceMadeEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	StoryID  uuid.UUID `json:"story_id"`
	SceneID  uuid.UUID `json:"scene_id"`
	ChoiceID uuid.UUID `json:"choice_id"`
}
