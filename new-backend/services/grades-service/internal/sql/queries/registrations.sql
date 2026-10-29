-- name: CreateRegistration :one
-- A row that is still there belongs to an earlier program of the same
-- student whose cancelled event has not been consumed yet: the new program
-- takes it over, so that event no longer removes the course.
INSERT INTO grades.student_course_registrations (
    student_id, course_id, semester, is_attendance_failed, program_id
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (student_id, course_id) DO UPDATE
SET program_id = EXCLUDED.program_id
RETURNING *;

-- name: DeleteProgramRegistrations :execrows
-- A registration that holds a score stays: deleting it would cascade the
-- score away. Enrollment refuses the change when one exists, so this only
-- matters for a score entered while the change was in flight.
DELETE FROM grades.student_course_registrations r
WHERE r.student_id = @student_id
  AND r.course_id = ANY(@course_ids::uuid[])
  AND (r.program_id = @program_id OR r.program_id IS NULL)
  AND NOT EXISTS (
      SELECT 1 FROM grades.student_assessment_scores sa WHERE sa.registration_id = r.id
  );

-- name: StudentHasGradesInSemester :one
SELECT (
    EXISTS (
        SELECT 1
        FROM grades.student_assessment_scores sa
        JOIN grades.student_course_registrations r ON r.id = sa.registration_id
        WHERE r.student_id = @student_id AND r.semester = @semester
    ) OR EXISTS (
        SELECT 1 FROM grades.student_completed_courses cc
        WHERE cc.student_id = @student_id AND cc.semester = @semester
    )
)::boolean AS has_grades;

-- name: GetRegistrationByID :one
SELECT
    r.*,
    s.student_number,
    s.first_name as student_first_name,
    s.last_name as student_last_name,
    s.department as student_department,
    c.course_code,
    c.course_name,
    c.instructor_id,
    c.assessment_schema
FROM grades.student_course_registrations r
JOIN grades.students_view s ON r.student_id = s.id
JOIN grades.courses_view c ON r.course_id = c.id
WHERE r.id = $1;

-- name: GetRegistrationsByIDs :many
SELECT
    r.*,
    s.student_number,
    s.first_name as student_first_name,
    s.last_name as student_last_name,
    s.department as student_department,
    c.course_code,
    c.course_name,
    c.instructor_id,
    c.assessment_schema
FROM grades.student_course_registrations r
JOIN grades.students_view s ON r.student_id = s.id
JOIN grades.courses_view c ON r.course_id = c.id
WHERE r.id = ANY($1::uuid[]);

-- name: GetRegistrationsByCourse :many
SELECT
    r.*,
    s.student_number,
    s.first_name as student_first_name,
    s.last_name as student_last_name,
    s.department as student_department
FROM grades.student_course_registrations r
JOIN grades.students_view s ON r.student_id = s.id
WHERE r.course_id = $1
ORDER BY s.student_number;

-- name: MarkAttendanceFailed :exec
UPDATE grades.student_course_registrations
SET is_attendance_failed = true
WHERE student_id = $1 AND course_id = $2;

-- name: CountRegistrationsByCourse :one
SELECT COUNT(*) FROM grades.student_course_registrations
WHERE course_id = $1;

-- name: CountEligibleRegistrationsByCourse :one
-- Registrations that still need a grade entered — attendance failures
-- are auto-scored as FF at finalize time and don't need scores.
SELECT COUNT(*) FROM grades.student_course_registrations
WHERE course_id = $1 AND COALESCE(is_attendance_failed, FALSE) = FALSE;

-- name: DeleteRegistrationsByCourse :exec
DELETE FROM grades.student_course_registrations WHERE course_id = $1;

-- name: GetActiveRegistrationsByStudent :many
-- Returns registrations the student has but has NOT yet been finalized into
-- grades.student_completed_courses, joined with course metadata and aggregated scores.
SELECT
    r.id AS registration_id,
    r.semester,
    c.course_code,
    c.course_name,
    c.credits,
    COALESCE(
        (
            SELECT jsonb_object_agg(
                sa.slug,
                jsonb_build_object(
                    'score', sa.score,
                    'is_absent', COALESCE(sa.is_absent, FALSE),
                    'is_locked', COALESCE(sa.is_locked, FALSE)
                )
            )
            FROM grades.student_assessment_scores sa
            WHERE sa.registration_id = r.id
        ),
        '{}'::jsonb
    ) AS scores
FROM grades.student_course_registrations r
JOIN grades.courses_view c ON r.course_id = c.id
WHERE r.student_id = $1
  AND NOT EXISTS (
      SELECT 1 FROM grades.student_completed_courses cc
      WHERE cc.student_id = r.student_id AND cc.course_id = r.course_id
  )
ORDER BY r.semester DESC, c.course_code;
