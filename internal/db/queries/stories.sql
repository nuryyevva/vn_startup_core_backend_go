-- name: ListPublishedStoriesPage :many
SELECT * FROM stories
WHERE is_published = true
  AND (sqlc.narg('genre')::text IS NULL OR genre = sqlc.narg('genre'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit_count') OFFSET sqlc.arg('offset_count');

-- name: CountPublishedStories :one
SELECT COUNT(*) FROM stories
WHERE is_published = true
  AND (sqlc.narg('genre')::text IS NULL OR genre = sqlc.narg('genre'));

-- name: GetStory :one
SELECT * FROM stories WHERE id = $1;

-- name: GetFirstSceneOfStory :one
SELECT * FROM scenes WHERE story_id = $1 ORDER BY order_index ASC LIMIT 1;

-- name: AddStoryBookmark :exec
INSERT INTO story_bookmarks (user_id, story_id) VALUES ($1, $2)
ON CONFLICT (user_id, story_id) DO NOTHING;

-- name: RemoveStoryBookmark :exec
DELETE FROM story_bookmarks WHERE user_id = $1 AND story_id = $2;

-- name: ListBookmarkedStories :many
SELECT s.* FROM story_bookmarks b
JOIN stories s ON s.id = b.story_id
WHERE b.user_id = $1
ORDER BY b.created_at DESC;
