package story

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
	"vn_startup_core_backend_go/pkg/events"
)

// Wallet is the subset of the wallet module's service the story module
// needs to charge for paid choices. Debit is expected to already return a
// well-formed *apperr.Error (e.g. insufficient_funds) on failure, which the
// story service simply forwards to the HTTP layer.
type Wallet interface {
	Debit(ctx context.Context, userID uuid.UUID, amount int32, reason string) error
}

const (
	walletReasonChoicePurchase = "choice_purchase"
	walletReasonSceneUnlock    = "scene_unlock"
)

type Service struct {
	repo      Repository
	wallet    Wallet
	publisher EventPublisher
	logger    *slog.Logger
}

func NewService(repo Repository, wallet Wallet, publisher EventPublisher, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, wallet: wallet, publisher: publisher, logger: logger}
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ListPublishedStories returns one page of the published catalog, optionally
// filtered by genre. page is 1-based; pageSize <= 0 falls back to
// defaultPageSize and is clamped to maxPageSize either way, so callers never
// need to validate these themselves.
func (s *Service) ListPublishedStories(ctx context.Context, genre *string, page, pageSize int32) (StoryPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	stories, total, err := s.repo.ListPublishedStories(ctx, ListStoriesFilter{
		Genre:  genre,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	})
	if err != nil {
		return StoryPage{}, err
	}

	return StoryPage{Stories: stories, Total: total, Page: page, PageSize: pageSize}, nil
}

// Bookmark adds storyID to userID's library. Idempotent: bookmarking an
// already-bookmarked story succeeds without creating a duplicate row.
func (s *Service) Bookmark(ctx context.Context, userID, storyID uuid.UUID) error {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		if errors.Is(err, ErrStoryNotFound) {
			return apperr.NotFound("story_not_found", "История не найдена")
		}
		return err
	}
	return s.repo.AddBookmark(ctx, userID, storyID)
}

// Unbookmark removes storyID from userID's library. Idempotent: unbookmarking
// a story that was never bookmarked succeeds without error.
func (s *Service) Unbookmark(ctx context.Context, userID, storyID uuid.UUID) error {
	return s.repo.RemoveBookmark(ctx, userID, storyID)
}

// GetPublishedStory returns a single published story by ID, or
// story_not_found if it doesn't exist or isn't published — unauthenticated
// callers (the story catalog is public) must never learn about unpublished
// stories through this lookup.
func (s *Service) GetPublishedStory(ctx context.Context, id uuid.UUID) (Story, error) {
	story, err := s.repo.GetStory(ctx, id)
	if err != nil {
		if errors.Is(err, ErrStoryNotFound) {
			return Story{}, apperr.NotFound("story_not_found", "История не найдена")
		}
		return Story{}, err
	}
	if !story.IsPublished {
		return Story{}, apperr.NotFound("story_not_found", "История не найдена")
	}
	return story, nil
}

// ListBookmarks returns userID's bookmarked stories, most recently
// bookmarked first.
func (s *Service) ListBookmarks(ctx context.Context, userID uuid.UUID) ([]Story, error) {
	return s.repo.ListBookmarkedStories(ctx, userID)
}

// CountFinishedStories returns how many stories userID has completed, using
// the same "on the last scene" rule as ProgressSummaryResponse.IsFinished —
// used by the achievement module.
func (s *Service) CountFinishedStories(ctx context.Context, userID uuid.UUID) (int32, error) {
	summaries, err := s.repo.ListProgressByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	var finished int32
	for _, p := range summaries {
		if p.CurrentOrderIndex >= p.TotalScenes-1 {
			finished++
		}
	}
	return finished, nil
}

// CountUnlockedScenes returns how many paid scenes userID has ever
// unlocked — used by the achievement module.
func (s *Service) CountUnlockedScenes(ctx context.Context, userID uuid.UUID) (int32, error) {
	unlocked, err := s.repo.ListUnlockedSceneIDs(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int32(len(unlocked)), nil
}

// GetOrCreateProgress returns the caller's progress on story, creating a
// fresh progress row pointed at the story's first scene if none exists yet.
func (s *Service) GetOrCreateProgress(ctx context.Context, userID, storyID uuid.UUID) (Progress, error) {
	progress, err := s.repo.GetProgress(ctx, userID, storyID)
	if err == nil {
		return progress, nil
	}
	if !errors.Is(err, ErrProgressNotFound) {
		return Progress{}, err
	}

	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		if errors.Is(err, ErrStoryNotFound) {
			return Progress{}, apperr.NotFound("story_not_found", "История не найдена")
		}
		return Progress{}, err
	}

	firstScene, err := s.repo.GetFirstScene(ctx, storyID)
	if err != nil {
		if errors.Is(err, ErrSceneNotFound) {
			return Progress{}, apperr.Internal("story_has_no_scenes", "У этой истории ещё нет сцен")
		}
		return Progress{}, err
	}

	return s.repo.CreateProgress(ctx, userID, storyID, firstScene.ID)
}

// ListMyProgress returns every story userID has made progress on, most
// recently updated first — the data source for the stories screen's
// "Continue Reading" section and the profile screen's "My Library" /
// "finished" stat, none of which can be derived from
// GetOrCreateProgress (it creates a row as a side effect, so calling it
// per-story would fabricate progress on stories the player never opened).
func (s *Service) ListMyProgress(ctx context.Context, userID uuid.UUID) ([]ProgressSummary, error) {
	return s.repo.ListProgressByUser(ctx, userID)
}

func (s *Service) GetScene(ctx context.Context, sceneID uuid.UUID) (Scene, []Choice, error) {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		if errors.Is(err, ErrSceneNotFound) {
			return Scene{}, nil, apperr.NotFound("scene_not_found", "Сцена не найдена")
		}
		return Scene{}, nil, err
	}

	choices, err := s.repo.ListChoices(ctx, sceneID)
	if err != nil {
		return Scene{}, nil, err
	}

	return scene, choices, nil
}

// GetSceneForPlayer is like GetScene but enforces the paid-unlock gate: if
// the scene has a non-nil UnlockCostDiamonds and userID hasn't unlocked it
// yet (via UnlockScene), it refuses to return the scene's content at all —
// unlocking is a precondition, not something inferred from the response.
func (s *Service) GetSceneForPlayer(ctx context.Context, userID, sceneID uuid.UUID) (Scene, []Choice, error) {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		if errors.Is(err, ErrSceneNotFound) {
			return Scene{}, nil, apperr.NotFound("scene_not_found", "Сцена не найдена")
		}
		return Scene{}, nil, err
	}

	if scene.UnlockCostDiamonds != nil {
		unlocked, err := s.repo.IsSceneUnlocked(ctx, userID, sceneID)
		if err != nil {
			return Scene{}, nil, err
		}
		if !unlocked {
			return Scene{}, nil, apperr.New(http.StatusPaymentRequired, "scene_locked", "Глава заблокирована, необходимо разблокировать за алмазы")
		}
	}

	choices, err := s.repo.ListChoices(ctx, sceneID)
	if err != nil {
		return Scene{}, nil, err
	}

	return scene, choices, nil
}

// ListStoryScenes returns every scene of storyID with a per-user lock flag,
// used to render the chapter list on the story detail screen.
func (s *Service) ListStoryScenes(ctx context.Context, userID, storyID uuid.UUID) ([]Scene, map[uuid.UUID]bool, error) {
	if _, err := s.repo.GetStory(ctx, storyID); err != nil {
		if errors.Is(err, ErrStoryNotFound) {
			return nil, nil, apperr.NotFound("story_not_found", "История не найдена")
		}
		return nil, nil, err
	}

	scenes, err := s.repo.ListScenesByStory(ctx, storyID)
	if err != nil {
		return nil, nil, err
	}

	unlockedSceneIDs, err := s.repo.ListUnlockedSceneIDs(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	return scenes, unlockedSceneIDs, nil
}

// UnlockScene charges userID scene.UnlockCostDiamonds and records the
// unlock, so a later GetSceneForPlayer call succeeds. It is idempotent:
// calling it again on an already-unlocked scene succeeds without a second
// charge. Scenes with no unlock cost can't be "unlocked" (there's nothing to
// pay for) and return scene_not_locked.
func (s *Service) UnlockScene(ctx context.Context, userID, sceneID uuid.UUID) error {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		if errors.Is(err, ErrSceneNotFound) {
			return apperr.NotFound("scene_not_found", "Сцена не найдена")
		}
		return err
	}

	if scene.UnlockCostDiamonds == nil {
		return apperr.BadRequest("scene_not_locked", "Эта глава не требует разблокировки")
	}

	alreadyUnlocked, err := s.repo.IsSceneUnlocked(ctx, userID, sceneID)
	if err != nil {
		return err
	}
	if alreadyUnlocked {
		return nil
	}

	if err := s.wallet.Debit(ctx, userID, *scene.UnlockCostDiamonds, walletReasonSceneUnlock); err != nil {
		return err
	}

	if err := s.repo.UnlockScene(ctx, userID, sceneID); err != nil {
		return err
	}

	return nil
}

// SubmitChoice validates and applies a player's choice on sceneID: charges
// diamonds if the choice is paid, advances (or keeps) progress, and
// publishes player.choice.made.
func (s *Service) SubmitChoice(ctx context.Context, userID, sceneID, choiceID uuid.UUID) (Progress, error) {
	scene, err := s.repo.GetScene(ctx, sceneID)
	if err != nil {
		if errors.Is(err, ErrSceneNotFound) {
			return Progress{}, apperr.NotFound("scene_not_found", "Сцена не найдена")
		}
		return Progress{}, err
	}

	choice, err := s.repo.GetChoice(ctx, choiceID)
	if err != nil {
		if errors.Is(err, ErrChoiceNotFound) {
			return Progress{}, apperr.NotFound("choice_not_found", "Вариант выбора не найден")
		}
		return Progress{}, err
	}
	if choice.SceneID != scene.ID {
		return Progress{}, apperr.BadRequest("choice_not_in_scene", "Этот вариант выбора не принадлежит указанной сцене")
	}

	if choice.IsPaid && choice.CostDiamonds > 0 {
		if err := s.wallet.Debit(ctx, userID, choice.CostDiamonds, walletReasonChoicePurchase); err != nil {
			return Progress{}, err
		}
	}

	progress, err := s.repo.GetProgress(ctx, userID, scene.StoryID)
	if err != nil {
		if !errors.Is(err, ErrProgressNotFound) {
			return Progress{}, err
		}
		progress, err = s.repo.CreateProgress(ctx, userID, scene.StoryID, scene.ID)
		if err != nil {
			return Progress{}, err
		}
	}

	nextSceneID := scene.ID
	if choice.NextSceneID != nil {
		nextSceneID = *choice.NextSceneID
	}
	choicesMade := append(append([]uuid.UUID{}, progress.ChoicesMade...), choice.ID)

	updated, err := s.repo.UpdateProgress(ctx, userID, scene.StoryID, nextSceneID, choicesMade)
	if err != nil {
		return Progress{}, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, events.SubjectPlayerChoiceMade, PlayerChoiceMadeEvent{
			UserID:   userID,
			StoryID:  scene.StoryID,
			SceneID:  scene.ID,
			ChoiceID: choice.ID,
		}); err != nil {
			s.logger.Error("publish player.choice.made failed", slog.String("error", err.Error()))
		}
	}

	return updated, nil
}
