package story

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrStoryNotFound    = errors.New("story not found")
	ErrSceneNotFound    = errors.New("scene not found")
	ErrChoiceNotFound   = errors.New("choice not found")
	ErrProgressNotFound = errors.New("progress not found")
)

type Repository interface {
	ListPublishedStories(ctx context.Context) ([]Story, error)
	GetStory(ctx context.Context, id uuid.UUID) (Story, error)
	GetFirstScene(ctx context.Context, storyID uuid.UUID) (Scene, error)
	GetScene(ctx context.Context, id uuid.UUID) (Scene, error)
	ListChoices(ctx context.Context, sceneID uuid.UUID) ([]Choice, error)
	GetChoice(ctx context.Context, id uuid.UUID) (Choice, error)
	GetProgress(ctx context.Context, userID, storyID uuid.UUID) (Progress, error)
	CreateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID) (Progress, error)
	UpdateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID, choicesMade []uuid.UUID) (Progress, error)
}
