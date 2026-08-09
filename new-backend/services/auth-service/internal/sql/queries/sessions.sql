-- name: CreateSession :one
INSERT INTO auth.sessions (user_id, refresh_token_jti, device_info, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, refresh_token_jti, device_info, ip_address, created_at, expires_at, last_used_at;

-- name: GetSessionByJTI :one
SELECT id, user_id, refresh_token_jti, device_info, ip_address, created_at, expires_at, last_used_at
FROM auth.sessions
WHERE refresh_token_jti = $1
LIMIT 1;

-- name: GetSessionsByUserID :many
SELECT id, user_id, refresh_token_jti, device_info, ip_address, created_at, expires_at, last_used_at
FROM auth.sessions
WHERE user_id = $1
ORDER BY last_used_at DESC;

-- name: UpdateSessionLastUsed :exec
UPDATE auth.sessions
SET last_used_at = NOW()
WHERE refresh_token_jti = $1;

-- name: DeleteSession :exec
DELETE FROM auth.sessions
WHERE refresh_token_jti = $1;

-- name: DeleteSessionByID :exec
DELETE FROM auth.sessions
WHERE id = $1 AND user_id = $2;

-- name: DeleteAllUserSessions :exec
DELETE FROM auth.sessions
WHERE user_id = $1;

-- name: CleanupExpiredSessions :exec
DELETE FROM auth.sessions
WHERE expires_at < NOW();
