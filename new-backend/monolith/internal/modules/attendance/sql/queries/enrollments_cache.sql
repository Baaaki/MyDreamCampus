-- name: CreateEnrollmentCache :exec
INSERT INTO attendance.enrollments_view (student_id, course_id, semester)
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
FROM attendance.enrollments_view e
JOIN attendance.students_view s ON e.student_id = s.id
WHERE e.course_id = $1 AND e.semester = $2;

-- name: CheckEnrollment :one
SELECT COUNT(*) as count
FROM attendance.enrollments_view
WHERE student_id = $1 AND course_id = $2 AND semester = $3;

-- name: GetStudentEnrollmentsBySemester :many
SELECT
    c.id as course_id,
    c.course_code,
    c.course_name,
    c.instructor_fullname,
    c.total_weeks,
    c.semester
FROM attendance.enrollments_view e
JOIN attendance.courses_view c ON e.course_id = c.id
WHERE e.student_id = $1 AND e.semester = $2;
