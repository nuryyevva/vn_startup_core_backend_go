-- name: GetScene :one
SELECT * FROM scenes WHERE id = $1;

-- name: ListScenesByStory :many
SELECT * FROM scenes WHERE story_id = $1 ORDER BY order_index ASC;

-- name: ListChoicesByScene :many
SELECT * FROM choices WHERE scene_id = $1 ORDER BY id ASC;

-- name: GetChoice :one
SELECT * FROM choices WHERE id = $1;

-- name: IsSceneUnlocked :one
SELECT EXISTS(SELECT 1 FROM scene_unlocks WHERE user_id = $1 AND scene_id = $2);

-- name: ListUnlockedSceneIDsByUser :many
SELECT scene_id FROM scene_unlocks WHERE user_id = $1;

-- name: CreateSceneUnlock :exec
INSERT INTO scene_unlocks (user_id, scene_id) VALUES ($1, $2)
ON CONFLICT (user_id, scene_id) DO NOTHING;
