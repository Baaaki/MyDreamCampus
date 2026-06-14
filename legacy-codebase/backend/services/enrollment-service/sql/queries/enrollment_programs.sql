-- name: CreateEnrollmentProgram :one
INSERT INTO enrollment_programs (
    student_id, semester, status, created_at
) VALUES (
    $1, $2, $3, NOW()
)
RETURNING id, student_id, semester, status, created_at;

-- name: GetEnrollmentProgramByID :one
SELECT id, student_id, semester, status, created_at
FROM enrollment_programs
WHERE id = $1
LIMIT 1;

-- name: GetEnrollmentProgramsByStudent :many
SELECT id, student_id, semester, status, created_at
FROM enrollment_programs
WHERE student_id = $1
  AND ($2::VARCHAR IS NULL OR $2 = '' OR semester = $2)
  AND ($3::VARCHAR IS NULL OR $3 = '' OR status = $3::enrollment_status_enum)
ORDER BY created_at DESC;

-- name: GetEnrollmentProgramByStudentAndSemester :one
SELECT id, student_id, semester, status, created_at
FROM enrollment_programs
WHERE student_id = $1 AND semester = $2
LIMIT 1;

-- name: GetPendingProgramsByAdvisor :many
SELECT 
    ep.id, 
    ep.student_id, 
    ep.semester, 
    ep.status, 
    ep.created_at,
    sc.first_name,
    sc.last_name,
    sc.student_number,
    sc.department,
    sc.class_level
FROM enrollment_programs ep
JOIN students_cache sc ON ep.student_id = sc.id
WHERE sc.advisor_id = $1 AND ep.status = 'pending'
ORDER BY ep.created_at;

-- name: UpdateProgramStatus :one
UPDATE enrollment_programs
SET status = $2
WHERE id = $1
RETURNING id, student_id, semester, status, created_at;

-- name: LockPendingProgram :one
SELECT id, student_id, semester, status, created_at
FROM enrollment_programs
WHERE id = $1 AND status = 'pending'
FOR UPDATE;

-- name: DeleteEnrollmentProgram :exec
DELETE FROM enrollment_programs
WHERE id = $1;
