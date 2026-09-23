package achievement

import (
	"context"

	"github.com/google/uuid"
)

// ActivityProvider assembles the cross-module snapshot achievements are
// evaluated against. Implemented in cmd/api/main.go by an adapter that reads
// from the story, wallet, dialog, and stats services — this package never
// imports any of them directly.
type ActivityProvider interface {
	GetActivityStats(ctx context.Context, userID uuid.UUID) (ActivityStats, error)
}

type Service struct {
	repo     Repository
	activity ActivityProvider
}

func NewService(repo Repository, activity ActivityProvider) *Service {
	return &Service{repo: repo, activity: activity}
}

// ListAchievements returns the full catalog for userID, unlocking (and
// persisting) any achievement whose condition is newly met by their current
// activity. Achievements are evaluated lazily on read rather than pushed
// from an event stream: simpler to reason about, and the profile screen is
// the only place they're shown, so "unlocked the moment you check" is an
// acceptable trade-off for this catalog's size.
func (s *Service) ListAchievements(ctx context.Context, userID uuid.UUID) ([]Achievement, error) {
	unlocked, err := s.repo.ListUnlocked(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats, err := s.activity.GetActivityStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]Achievement, 0, len(Catalog))
	for _, def := range Catalog {
		if unlockedAt, ok := unlocked[def.ID]; ok {
			at := unlockedAt
			out = append(out, Achievement{ID: def.ID, Title: def.Title, Description: def.Description, Unlocked: true, UnlockedAt: &at})
			continue
		}

		if !def.Check(stats) {
			out = append(out, Achievement{ID: def.ID, Title: def.Title, Description: def.Description, Unlocked: false})
			continue
		}

		unlockedAt, err := s.repo.Unlock(ctx, userID, def.ID)
		if err != nil {
			return nil, err
		}
		at := unlockedAt
		out = append(out, Achievement{ID: def.ID, Title: def.Title, Description: def.Description, Unlocked: true, UnlockedAt: &at})
	}

	return out, nil
}
