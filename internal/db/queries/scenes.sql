-- name: GetScene :one
-- With the story's coming_soon, so a scene of a story that can't be played
-- yet is refused without another round trip.
SELECT sqlc.embed(scenes), stories.coming_soon AS story_coming_soon
FROM scenes
JOIN stories ON stories.id = scenes.story_id
WHERE scenes.id = $1;

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

-- name: ListChoiceIDsFromEarlierScenes :many
-- Which of the given choices belong to scenes that come before order_index
-- in the story.
SELECT c.id FROM choices c
JOIN scenes s ON s.id = c.scene_id
WHERE s.story_id = sqlc.arg(story_id)
  AND s.order_index < sqlc.arg(order_index)
  AND c.id = ANY(sqlc.arg(choice_ids)::uuid[]);

-- name: ListChoicesByOnceKeys :many
SELECT c.id, c.once_key FROM choices c
JOIN scenes s ON s.id = c.scene_id
WHERE s.story_id = sqlc.arg(story_id)
  AND c.once_key = ANY(sqlc.arg(once_keys)::text[]);
