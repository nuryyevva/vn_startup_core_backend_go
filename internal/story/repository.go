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

// ListStoriesFilter narrows and paginates ListPublishedStories: Genre nil
// means no genre filter, Limit/Offset are already resolved (defaulted and
// clamped) by the service before reaching the repository.
type ListStoriesFilter struct {
	Genre  *string
	Limit  int32
	Offset int32
}

type Repository interface {
	ListPublishedStories(ctx context.Context, filter ListStoriesFilter) ([]Story, int64, error)
	GetStory(ctx context.Context, id uuid.UUID) (Story, error)
	GetFirstScene(ctx context.Context, storyID uuid.UUID) (Scene, error)
	GetScene(ctx context.Context, id uuid.UUID) (Scene, error)
	ListScenesByStory(ctx context.Context, storyID uuid.UUID) ([]Scene, error)
	ListChoices(ctx context.Context, sceneID uuid.UUID) ([]Choice, error)
	GetChoice(ctx context.Context, id uuid.UUID) (Choice, error)
	GetProgress(ctx context.Context, userID, storyID uuid.UUID) (Progress, error)
	ListProgressByUser(ctx context.Context, userID uuid.UUID) ([]ProgressSummary, error)
	CreateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID) (Progress, error)
	UpdateProgress(ctx context.Context, userID, storyID, sceneID uuid.UUID, choicesMade []uuid.UUID) (Progress, error)
	IsSceneUnlocked(ctx context.Context, userID, sceneID uuid.UUID) (bool, error)
	ListUnlockedSceneIDs(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error)
	UnlockScene(ctx context.Context, userID, sceneID uuid.UUID) error
	AddBookmark(ctx context.Context, userID, storyID uuid.UUID) error
	RemoveBookmark(ctx context.Context, userID, storyID uuid.UUID) error
	ListBookmarkedStories(ctx context.Context, userID uuid.UUID) ([]Story, error)
}
