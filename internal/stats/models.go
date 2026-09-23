package stats

import (
	"github.com/google/uuid"
)

// Stats is a user's reading-activity summary: cumulative reading time and
// day-streak, both derived entirely from heartbeat calls (see Service) —
// there is no other source of truth for either.
type Stats struct {
	UserID              uuid.UUID
	TotalReadingSeconds int64
	DayStreak           int32
	LongestStreak       int32
}

// HeartbeatRequest reports secondsElapsed of active reading time since the
// caller's last heartbeat. The client is expected to call this periodically
// (e.g. every ~30s) while a scene is on screen.
type HeartbeatRequest struct {
	Seconds int32 `json:"seconds"`
}

type StatsResponse struct {
	DayStreak           int32   `json:"day_streak"`
	LongestStreak       int32   `json:"longest_streak"`
	TotalReadingSeconds int64   `json:"total_reading_seconds"`
	HoursRead           float64 `json:"hours_read"`
}

func toStatsResponse(s Stats) StatsResponse {
	return StatsResponse{
		DayStreak:           s.DayStreak,
		LongestStreak:       s.LongestStreak,
		TotalReadingSeconds: s.TotalReadingSeconds,
		HoursRead:           roundToOneDecimal(float64(s.TotalReadingSeconds) / 3600),
	}
}

func roundToOneDecimal(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

// zeroStats is returned for a user who has never sent a heartbeat yet —
// GetStats must not error just because user_stats has no row for them.
func zeroStats(userID uuid.UUID) Stats {
	return Stats{UserID: userID}
}
