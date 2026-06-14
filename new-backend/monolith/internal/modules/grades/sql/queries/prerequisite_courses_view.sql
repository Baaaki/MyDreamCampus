-- name: IsPrerequisiteCourse :one
SELECT EXISTS(
    SELECT 1 FROM grades.prerequisite_courses_view
    WHERE course_code = $1
) as is_prerequisite;
