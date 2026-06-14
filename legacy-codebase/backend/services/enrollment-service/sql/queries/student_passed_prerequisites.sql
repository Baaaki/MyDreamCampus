-- name: CheckPrerequisitePassed :one
SELECT EXISTS(
    SELECT 1 FROM student_passed_prerequisites
    WHERE student_id = $1 AND course_code = $2
) as passed;

-- name: UpsertPassedPrerequisite :one
INSERT INTO student_passed_prerequisites (
    student_id, course_code, semester, grade_point, synced_at
) VALUES (
    $1, $2, $3, $4, NOW()
)
ON CONFLICT (student_id, course_code) DO UPDATE SET
    semester = EXCLUDED.semester,
    grade_point = EXCLUDED.grade_point,
    synced_at = NOW()
RETURNING student_id, course_code, semester, grade_point, synced_at;

-- name: GetPassedPrerequisitesByStudent :many
SELECT student_id, course_code, semester, grade_point, synced_at
FROM student_passed_prerequisites
WHERE student_id = $1
ORDER BY semester DESC;
