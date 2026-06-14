-- name: CreateSemester :one
INSERT INTO semesters (name, hard_deadline)
VALUES ($1, $2)
RETURNING *;

-- name: GetSemesterByName :one
SELECT * FROM semesters WHERE name = $1;

-- name: GetActiveSemester :one
SELECT * FROM semesters WHERE status = 'active' LIMIT 1;

-- name: ListSemesters :many
SELECT * FROM semesters ORDER BY created_at DESC;

-- name: ActivateSemester :one
UPDATE semesters SET status = 'active' WHERE id = $1 AND status = 'planned'
RETURNING *;

-- name: CompleteSemester :one
UPDATE semesters SET status = 'completed' WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: AutoCompleteSemester :exec
UPDATE semesters SET status = 'completed'
WHERE name = $1 AND status = 'active' AND hard_deadline < NOW();

-- name: HasActiveSemester :one
-- INVARIANT: Only one semester can be active at any given time.
-- Used before activation to give a clear error message at the application layer.
-- The database also enforces this via idx_semesters_single_active partial unique index.
SELECT EXISTS(SELECT 1 FROM semesters WHERE status = 'active') AS has_active;

-- name: GetSemesterByID :one
SELECT * FROM semesters WHERE id = $1;

-- name: DeletePlannedSemester :exec
DELETE FROM semesters WHERE id = $1 AND status = 'planned';

-- name: DeleteSemesterCoursesBySemester :exec
DELETE FROM semester_courses WHERE semester = $1;

-- name: UpdatePlannedSemester :one
UPDATE semesters
SET hard_deadline = $2, updated_at = NOW()
WHERE id = $1 AND status = 'planned'
RETURNING *;
