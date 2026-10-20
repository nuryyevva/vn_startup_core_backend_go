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

// PlayerStoryRestartedEvent is published to NATS subject "player.story.restarted" when a player starts a story
// over (or from a chapter): the AI module drops its memories of that player for the story's characters.
type PlayerStoryRestartedEvent struct {
	UserID  uuid.UUID `json:"user_id"`
	StoryID uuid.UUID `json:"story_id"`
	// FromOrder is set when the player restarted from a chapter: the order_index of the scene they went back to.
	// The AI module forgets only what happened at or after it. Nil means the whole story was restarted.
	FromOrder *int32 `json:"from_order,omitempty"`
}

// PlayerChoiceMadeEvent is published to NATS subject "player.choice.made"
// whenever a player submits a choice.
type PlayerChoiceMadeEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	StoryID  uuid.UUID `json:"story_id"`
	SceneID  uuid.UUID `json:"scene_id"`
	ChoiceID uuid.UUID `json:"choice_id"`
}
