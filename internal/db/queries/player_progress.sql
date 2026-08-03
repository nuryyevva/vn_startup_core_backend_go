-- name: GetPlayerProgress :one
SELECT * FROM player_progress WHERE user_id = $1 AND story_id = $2;

-- name: CreatePlayerProgress :one
INSERT INTO player_progress (user_id, story_id, current_scene_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdatePlayerProgress :one
UPDATE player_progress
SET current_scene_id = $3,
    choices_made = $4,
    updated_at = now()
WHERE user_id = $1 AND story_id = $2
RETURNING *;
