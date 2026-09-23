-- name: GetUserStats :one
SELECT * FROM user_stats WHERE user_id = $1;

-- name: UpsertUserStatsHeartbeat :one
INSERT INTO user_stats (user_id, total_reading_seconds, day_streak, longest_streak, last_active_date, updated_at)
VALUES (sqlc.arg('user_id'), sqlc.arg('seconds'), 1, 1, sqlc.arg('today'), now())
ON CONFLICT (user_id) DO UPDATE SET
    total_reading_seconds = user_stats.total_reading_seconds + EXCLUDED.total_reading_seconds,
    day_streak = CASE
        WHEN user_stats.last_active_date = EXCLUDED.last_active_date THEN user_stats.day_streak
        WHEN user_stats.last_active_date = EXCLUDED.last_active_date - 1 THEN user_stats.day_streak + 1
        ELSE 1
    END,
    longest_streak = GREATEST(
        user_stats.longest_streak,
        CASE
            WHEN user_stats.last_active_date = EXCLUDED.last_active_date THEN user_stats.day_streak
            WHEN user_stats.last_active_date = EXCLUDED.last_active_date - 1 THEN user_stats.day_streak + 1
            ELSE 1
        END
    ),
    last_active_date = EXCLUDED.last_active_date,
    updated_at = now()
RETURNING *;
