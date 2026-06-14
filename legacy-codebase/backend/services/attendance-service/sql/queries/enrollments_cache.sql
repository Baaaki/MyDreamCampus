-- name: CreateEnrollmentCache :exec
INSERT INTO enrollments_cache (student_id, course_id, semester)
VALUES ($1, $2, $3)
ON CONFLICT (student_id, course_id) DO NOTHING;

-- name: GetEnrolledStudentsByCourse :many
SELECT
    s.id,
    s.student_number,
    s.first_name,
    s.last_name,
    s.email,
    s.department,
    s.is_active
FROM enrollments_cache e
JOIN students_cache s ON e.student_id = s.id
WHERE e.course_id = $1 AND e.semester = $2;

-- name: CheckEnrollment :one
SELECT COUNT(*) as count
FROM enrollments_cache
WHERE student_id = $1 AND course_id = $2 AND semester = $3;

-- name: GetStudentEnrollmentsBySemester :many
SELECT
    c.id as course_id,
    c.course_code,
    c.course_name,
    c.instructor_fullname,
    c.total_weeks,
    c.semester
FROM enrollments_cache e
JOIN courses_cache c ON e.course_id = c.id
WHERE e.student_id = $1 AND e.semester = $2;
