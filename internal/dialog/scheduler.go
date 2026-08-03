package dialog

import (
	"context"
	"log/slog"
	"time"
)

// SessionScheduler periodically ends active time-limited sessions whose
// deadline has passed, so a client that never sends another message (or
// never calls /end) doesn't leave a session "active" forever.
type SessionScheduler struct {
	service  *Service
	interval time.Duration
	logger   *slog.Logger
}

func NewSessionScheduler(service *Service, interval time.Duration, logger *slog.Logger) *SessionScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SessionScheduler{service: service, interval: interval, logger: logger}
}

// Run blocks, ticking every interval until ctx is cancelled. Call it in its
// own goroutine from main.go.
func (s *SessionScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ended, err := s.service.EndExpiredSessions(ctx)
			if err != nil {
				s.logger.Error("session scheduler tick failed", slog.String("error", err.Error()))
				continue
			}
			if ended > 0 {
				s.logger.Info("session scheduler ended expired sessions", slog.Int("count", ended))
			}
		}
	}
}
