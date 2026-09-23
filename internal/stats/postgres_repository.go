package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/db/sqlc"
)

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(pool)}
}

func (r *PostgresRepository) GetStats(ctx context.Context, userID uuid.UUID) (Stats, error) {
	row, err := r.queries.GetUserStats(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stats{}, ErrStatsNotFound
		}
		return Stats{}, fmt.Errorf("get user stats: %w", err)
	}
	return toDomainStats(row), nil
}

func (r *PostgresRepository) RecordHeartbeat(ctx context.Context, userID uuid.UUID, seconds int32, today time.Time) (Stats, error) {
	row, err := r.queries.UpsertUserStatsHeartbeat(ctx, sqlc.UpsertUserStatsHeartbeatParams{
		UserID:  userID,
		Seconds: int64(seconds),
		Today:   pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		return Stats{}, fmt.Errorf("upsert user stats heartbeat: %w", err)
	}
	return toDomainStats(row), nil
}

func toDomainStats(s sqlc.UserStat) Stats {
	return Stats{
		UserID:              s.UserID,
		TotalReadingSeconds: s.TotalReadingSeconds,
		DayStreak:           s.DayStreak,
		LongestStreak:       s.LongestStreak,
	}
}
