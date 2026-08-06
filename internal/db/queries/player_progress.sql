-- name: GetPlayerProgress :one
SELECT * FROM player_progress WHERE user_id = $1 AND story_id = $2;

-- name: ListPlayerProgressByUser :many
SELECT
    s.id AS story_id,
    s.title AS story_title,
    s.description AS story_description,
    s.cover_url AS story_cover_url,
    s.genre AS story_genre,
    s.status AS story_status,
    s.created_at AS story_created_at,
    pp.current_scene_id,
    pp.choices_made,
    pp.updated_at,
    sc.order_index AS current_order_index,
    (SELECT COUNT(*) FROM scenes WHERE scenes.story_id = s.id)::int AS total_scenes
FROM player_progress pp
JOIN stories s ON s.id = pp.story_id
JOIN scenes sc ON sc.id = pp.current_scene_id
WHERE pp.user_id = $1
ORDER BY pp.updated_at DESC;

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
