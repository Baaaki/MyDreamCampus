-- name: CreatePeriod :one
INSERT INTO academic_periods (semester, course_id, period_start, period_end, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, semester, course_id, period_start, period_end, is_active, created_at, updated_at;

-- name: ListAllPeriods :many
SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
FROM academic_periods
ORDER BY semester DESC, created_at DESC;

-- name: ListPeriodsBySemester :many
SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
FROM academic_periods
WHERE semester = $1
ORDER BY created_at DESC;

-- name: GetPeriodByID :one
SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
FROM academic_periods
WHERE id = $1;

-- name: UpdatePeriod :one
UPDATE academic_periods
SET period_end = COALESCE(sqlc.narg('period_end'), period_end),
    is_active  = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, semester, course_id, period_start, period_end, is_active, created_at, updated_at;

-- name: DeletePeriod :exec
DELETE FROM academic_periods WHERE id = $1;

-- name: GetCourseSpecificPeriod :one
SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
FROM academic_periods
WHERE semester = $1 AND course_id = $2 AND is_active = true
LIMIT 1;

-- name: GetGlobalPeriod :one
SELECT id, semester, course_id, period_start, period_end, is_active, created_at, updated_at
FROM academic_periods
WHERE semester = $1 AND course_id IS NULL AND is_active = true
LIMIT 1;
