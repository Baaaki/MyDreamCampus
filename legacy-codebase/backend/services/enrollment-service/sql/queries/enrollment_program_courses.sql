-- name: CreateEnrollmentProgramCourse :one
INSERT INTO enrollment_program_courses (
    program_id, course_id, created_at
) VALUES (
    $1, $2, NOW()
)
RETURNING id, program_id, course_id, created_at;

-- name: GetCoursesByProgramID :many
SELECT epc.id, epc.program_id, epc.course_id, epc.created_at,
       sc.course_code, sc.course_name, sc.credits, sc.instructor_fullname
FROM enrollment_program_courses epc
JOIN semester_courses_cache sc ON epc.course_id = sc.id
WHERE epc.program_id = $1
ORDER BY sc.course_code;

-- name: DeleteProgramCourses :exec
DELETE FROM enrollment_program_courses
WHERE program_id = $1;
