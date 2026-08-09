-- name: IsPrerequisiteCourse :one
SELECT EXISTS(
    SELECT 1 FROM grades.prerequisite_courses_view
    WHERE course_code = $1
) as is_prerequisite;

-- name: UpsertPrerequisiteCourse :exec
INSERT INTO grades.prerequisite_courses_view (course_code, course_id, synced_at)
VALUES ($1, $2, NOW())
ON CONFLICT (course_code, course_id) DO UPDATE SET synced_at = NOW();
