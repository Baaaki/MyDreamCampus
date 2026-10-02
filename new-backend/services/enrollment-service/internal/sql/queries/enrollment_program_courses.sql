-- name: CreateEnrollmentProgramCourse :one
INSERT INTO enrollment.enrollment_program_courses (
    program_id, course_id, course_code, course_name, credits, created_at
) VALUES (
    $1, $2, $3, $4, $5, NOW()
)
RETURNING id, program_id, course_id, course_code, course_name, credits, created_at;

-- name: GetCoursesByProgramID :many
SELECT id, program_id, course_id, course_code, course_name, credits, created_at
FROM enrollment.enrollment_program_courses
WHERE program_id = $1
ORDER BY course_code;

-- name: DeleteProgramCourses :exec
DELETE FROM enrollment.enrollment_program_courses
WHERE program_id = $1;

-- name: LockCourseSeats :exec
-- Holds one course's seat count still until the calling transaction ends.
-- Callers take these in ascending lock_id order, so two submissions that
-- want the same courses never wait on each other in a circle.
SELECT pg_advisory_xact_lock(sqlc.arg(lock_id)::bigint);

-- name: CountActiveCourseEnrollments :one
-- Seats taken on a course: a pending program holds its seat until it is
-- rejected or cancelled, an approved one for good.
SELECT COUNT(*)
FROM enrollment.enrollment_program_courses epc
JOIN enrollment.enrollment_programs ep ON ep.id = epc.program_id
WHERE epc.course_id = $1 AND ep.status IN ('pending', 'approved');
