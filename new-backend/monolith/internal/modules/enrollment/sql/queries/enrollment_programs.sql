-- name: CreateEnrollmentProgram :one
INSERT INTO enrollment.enrollment_programs (
    student_id, semester, status, created_at
) VALUES (
    $1, $2, $3, NOW()
)
RETURNING id, student_id, semester, status, created_at;

-- name: GetEnrollmentProgramByID :one
SELECT id, student_id, semester, status, created_at
FROM enrollment.enrollment_programs
WHERE id = $1
LIMIT 1;

-- name: GetEnrollmentProgramsByStudent :many
SELECT id, student_id, semester, status, created_at
FROM enrollment.enrollment_programs
WHERE student_id = $1
  AND ($2::VARCHAR IS NULL OR $2 = '' OR semester = $2)
  AND ($3::VARCHAR IS NULL OR $3 = '' OR status = $3::enrollment.enrollment_status_enum)
ORDER BY created_at DESC;

-- name: GetEnrollmentProgramByStudentAndSemester :one
SELECT id, student_id, semester, status, created_at
FROM enrollment.enrollment_programs
WHERE student_id = $1 AND semester = $2
LIMIT 1;

-- name: GetPendingProgramsByStudentIDs :many
SELECT id, student_id, semester, status, created_at
FROM enrollment.enrollment_programs
WHERE status = 'pending' AND student_id = ANY($1::uuid[])
ORDER BY created_at;

-- name: UpdateProgramStatus :one
UPDATE enrollment.enrollment_programs
SET status = $2
WHERE id = $1
RETURNING id, student_id, semester, status, created_at;

-- name: LockPendingProgram :one
SELECT id, student_id, semester, status, created_at
FROM enrollment.enrollment_programs
WHERE id = $1 AND status = 'pending'
FOR UPDATE;

-- name: DeleteEnrollmentProgram :exec
DELETE FROM enrollment.enrollment_programs
WHERE id = $1;
