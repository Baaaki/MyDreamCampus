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
