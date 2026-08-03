-- name: CreateDialogSession :one
INSERT INTO dialog_sessions (user_id, character_id, scene_id, limit_type, limit_value)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDialogSession :one
SELECT * FROM dialog_sessions WHERE id = $1;

-- name: GetDialogSessionForUpdate :one
SELECT * FROM dialog_sessions WHERE id = $1 FOR UPDATE;

-- name: IncrementDialogSessionMessageCount :one
UPDATE dialog_sessions
SET message_count = message_count + 1
WHERE id = $1
RETURNING *;

-- name: EndDialogSession :one
UPDATE dialog_sessions
SET status = 'ended',
    ended_at = now(),
    end_reason = $2
WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: ListExpiredTimeLimitedSessions :many
SELECT * FROM dialog_sessions
WHERE status = 'active'
  AND limit_type = 'time'
  AND limit_value IS NOT NULL
  AND started_at + make_interval(secs => limit_value) <= now();

-- name: CreateDialogMessage :one
INSERT INTO dialog_messages (session_id, sender, text, cost_diamonds)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDialogMessages :many
SELECT * FROM dialog_messages WHERE session_id = $1 ORDER BY created_at ASC;
