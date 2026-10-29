-- name: CreateEnrollmentCache :exec
-- A row that is still there belongs to an earlier program of the same
-- student whose cancelled event has not been consumed yet: the new program
-- takes it over, so that event no longer removes the course.
INSERT INTO attendance.enrollments_view (student_id, course_id, semester, enrolled_at, program_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (student_id, course_id) DO UPDATE
SET semester = EXCLUDED.semester,
    enrolled_at = EXCLUDED.enrolled_at,
    program_id = EXCLUDED.program_id;

-- name: DeleteProgramEnrollments :execrows
DELETE FROM attendance.enrollments_view
WHERE student_id = @student_id
  AND course_id = ANY(@course_ids::uuid[])
  AND (program_id = @program_id OR program_id IS NULL);

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
    c.semester,
    e.enrolled_at
FROM attendance.enrollments_view e
JOIN attendance.courses_view c ON e.course_id = c.id
WHERE e.student_id = $1 AND e.semester = $2;
