package stats

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrStatsNotFound = errors.New("user stats not found")

type Repository interface {
	GetStats(ctx context.Context, userID uuid.UUID) (Stats, error)
	// RecordHeartbeat atomically adds seconds to userID's total reading time
	// and updates their day-streak against today (see the SQL query for the
	// exact streak rules: same day is a no-op, the very next day extends the
	// streak, any bigger gap resets it to 1).
	RecordHeartbeat(ctx context.Context, userID uuid.UUID, seconds int32, today time.Time) (Stats, error)
}
