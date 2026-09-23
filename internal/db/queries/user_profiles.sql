-- name: CreateUserProfile :one
INSERT INTO user_profiles (user_id)
VALUES ($1)
RETURNING *;

-- name: GetUserProfile :one
SELECT * FROM user_profiles WHERE user_id = $1;

-- name: GetUserWithProfile :one
SELECT u.id, u.email, u.role, u.created_at, p.display_name, p.gender, p.favorite_genres, p.theme, p.language,
       p.notify_new_chapters, p.notify_promotional_offers, p.updated_at
FROM users u
JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1;

-- name: UpdateUserProfile :one
UPDATE user_profiles
SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    gender = COALESCE(sqlc.narg('gender'), gender),
    favorite_genres = COALESCE(sqlc.narg('favorite_genres'), favorite_genres),
    theme = COALESCE(sqlc.narg('theme'), theme),
    language = COALESCE(sqlc.narg('language'), language),
    notify_new_chapters = COALESCE(sqlc.narg('notify_new_chapters'), notify_new_chapters),
    notify_promotional_offers = COALESCE(sqlc.narg('notify_promotional_offers'), notify_promotional_offers),
    updated_at = now()
WHERE user_id = sqlc.arg('user_id')
RETURNING *;
