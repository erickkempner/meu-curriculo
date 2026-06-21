-- name: CreateSession :one
INSERT INTO sessions (user_id, token, csrf_token, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindSessionByToken :one
SELECT * FROM sessions WHERE token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= NOW();
