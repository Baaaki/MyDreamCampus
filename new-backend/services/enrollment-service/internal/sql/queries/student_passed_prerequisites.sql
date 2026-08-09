-- name: UpsertPassedPrerequisite :exec
INSERT INTO enrollment.student_passed_prerequisites (student_id, course_id, course_code, semester, grade_point, synced_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (student_id, course_code) DO UPDATE SET
    course_id = EXCLUDED.course_id,
    semester = EXCLUDED.semester,
    grade_point = EXCLUDED.grade_point,
    synced_at = NOW();

-- name: HasPassedPrerequisite :one
SELECT EXISTS(
    SELECT 1 FROM enrollment.student_passed_prerequisites
    WHERE student_id = $1 AND course_code = $2
) as passed;
