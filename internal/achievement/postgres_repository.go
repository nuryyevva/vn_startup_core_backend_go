package achievement

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"vn_startup_core_backend_go/internal/db/sqlc"
)

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: sqlc.New(pool)}
}

func (r *PostgresRepository) ListUnlocked(ctx context.Context, userID uuid.UUID) (map[string]time.Time, error) {
	rows, err := r.queries.ListUnlockedAchievements(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list unlocked achievements: %w", err)
	}
	out := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		out[row.AchievementID] = row.UnlockedAt
	}
	return out, nil
}

func (r *PostgresRepository) Unlock(ctx context.Context, userID uuid.UUID, achievementID string) (time.Time, error) {
	row, err := r.queries.UnlockAchievement(ctx, sqlc.UnlockAchievementParams{
		UserID:        userID,
		AchievementID: achievementID,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("unlock achievement: %w", err)
	}
	return row.UnlockedAt, nil
}
