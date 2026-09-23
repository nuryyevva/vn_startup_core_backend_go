package stats

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"vn_startup_core_backend_go/pkg/apperr"
)

// maxHeartbeatSeconds caps a single heartbeat call so a client can't inflate
// total_reading_seconds by sending an inflated value — the frontend is
// expected to call in with the real elapsed time on a short interval
// (e.g. ~30s), so anything larger than this is rejected outright rather
// than silently clamped, since it indicates a bug or abuse rather than a
// legitimate slow heartbeat.
const maxHeartbeatSeconds = 300

type Service struct {
	repo Repository
	// now is overridable in tests; production code always uses time.Now.
	now func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// GetStats returns userID's reading stats, or all-zero stats if they've
// never sent a heartbeat yet (not an error — every user has stats, they're
// just empty until the first heartbeat).
func (s *Service) GetStats(ctx context.Context, userID uuid.UUID) (Stats, error) {
	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrStatsNotFound) {
			return zeroStats(userID), nil
		}
		return Stats{}, err
	}
	return stats, nil
}

// Heartbeat records secondsElapsed of active reading time for userID and
// returns the updated stats, so the caller can render fresh values
// immediately without a second round trip.
func (s *Service) Heartbeat(ctx context.Context, userID uuid.UUID, secondsElapsed int32) (Stats, error) {
	if secondsElapsed <= 0 || secondsElapsed > maxHeartbeatSeconds {
		return Stats{}, apperr.BadRequest("invalid_amount", "Некорректная длительность чтения")
	}

	today := s.now().UTC().Truncate(24 * time.Hour)
	return s.repo.RecordHeartbeat(ctx, userID, secondsElapsed, today)
}
