-- name: ListUnlockedAchievements :many
SELECT * FROM user_achievements WHERE user_id = $1;

-- name: UnlockAchievement :one
INSERT INTO user_achievements (user_id, achievement_id)
VALUES ($1, $2)
ON CONFLICT (user_id, achievement_id) DO UPDATE SET achievement_id = user_achievements.achievement_id
RETURNING *;
