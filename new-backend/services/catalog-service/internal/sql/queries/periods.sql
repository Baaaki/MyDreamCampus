-- academic_periods now holds one row per period_type, not one per semester.
-- Only DeletePeriodsBySemester below is still called (it clears every type for
-- a semester, which is what semester deletion wants). Every other query here
-- is unused and NOT type-scoped — reading through one would return an
-- arbitrary service's period. Use SimplePeriodRepository, which scopes by
-- type, or add the period_type predicate before wiring one of these up.

-- name: CreatePeriod :one
INSERT INTO course_catalog.academic_periods (semester, period_start, period_end, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at;

-- name: ListAllPeriods :many
SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
FROM course_catalog.academic_periods
ORDER BY semester DESC, created_at DESC;

-- name: ListPeriodsBySemester :many
SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
FROM course_catalog.academic_periods
WHERE semester = $1
ORDER BY created_at DESC;

-- name: GetPeriodByID :one
SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
FROM course_catalog.academic_periods
WHERE id = $1;

-- name: UpdatePeriod :one
UPDATE course_catalog.academic_periods
SET period_end = COALESCE(sqlc.narg('period_end'), period_end),
    is_active  = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, semester, period_start, period_end, is_active, created_at, updated_at;

-- name: DeletePeriod :exec
DELETE FROM course_catalog.academic_periods WHERE id = $1;

-- name: GetActivePeriodBySemester :one
SELECT id, semester, period_start, period_end, is_active, created_at, updated_at
FROM course_catalog.academic_periods
WHERE semester = $1 AND is_active = true
LIMIT 1;

-- name: DeletePeriodsBySemester :exec
DELETE FROM course_catalog.academic_periods WHERE semester = $1;

-- name: GetPeriodBySemester :one
SELECT * FROM course_catalog.academic_periods WHERE semester = $1 LIMIT 1;

-- name: UpdatePeriodBySemesterSQL :one
UPDATE course_catalog.academic_periods
SET period_start = $2, period_end = $3, updated_at = NOW()
WHERE semester = $1
RETURNING *;
