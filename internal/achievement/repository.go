package achievement

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	// ListUnlocked returns achievement_id -> unlocked_at for every
	// achievement userID has already unlocked.
	ListUnlocked(ctx context.Context, userID uuid.UUID) (map[string]time.Time, error)
	// Unlock idempotently records achievementID as unlocked for userID and
	// returns the unlock timestamp (the original one, if it was already
	// unlocked — calling this twice never moves UnlockedAt forward).
	Unlock(ctx context.Context, userID uuid.UUID, achievementID string) (time.Time, error)
}
