-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, department, is_active, token_version,
       force_password_change, failed_login_attempts, locked_until,
       created_at, updated_at, deleted_at
FROM users
WHERE email = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, department, is_active, token_version,
       force_password_change, failed_login_attempts, locked_until,
       created_at, updated_at, deleted_at
FROM users
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, role, department, is_active, token_version, force_password_change)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING
RETURNING id, email, role, department, is_active, force_password_change, created_at, updated_at;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2,
    force_password_change = $3,
    token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateUser :exec
UPDATE users
SET email = COALESCE(sqlc.narg('email'), email),
    department = COALESCE(sqlc.narg('department'), department),
    updated_at = NOW()
WHERE id = sqlc.arg('id');

-- name: IncrementTokenVersion :one
UPDATE users
SET token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING token_version;

-- name: IncrementFailedLoginAttempts :exec
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: ResetFailedLoginAttempts :exec
UPDATE users
SET failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $1;

-- name: LockAccount :exec
UPDATE users
SET locked_until = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false,
    deleted_at = NOW(),
    token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: AdminExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin' AND is_active = true) AS exists;

-- name: CheckEmailVersionSync :one
UPDATE users
SET token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1 AND email != $2
RETURNING token_version;
