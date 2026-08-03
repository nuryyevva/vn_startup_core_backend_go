-- name: GetScene :one
SELECT * FROM scenes WHERE id = $1;

-- name: ListChoicesByScene :many
SELECT * FROM choices WHERE scene_id = $1 ORDER BY id ASC;

-- name: GetChoice :one
SELECT * FROM choices WHERE id = $1;
