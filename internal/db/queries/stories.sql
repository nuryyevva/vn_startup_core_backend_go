-- name: ListPublishedStories :many
SELECT * FROM stories WHERE is_published = true ORDER BY created_at DESC;

-- name: GetStory :one
SELECT * FROM stories WHERE id = $1;

-- name: GetFirstSceneOfStory :one
SELECT * FROM scenes WHERE story_id = $1 ORDER BY order_index ASC LIMIT 1;
