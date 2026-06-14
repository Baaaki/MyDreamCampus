-- name: IsPrerequisiteCourse :one
SELECT EXISTS(
    SELECT 1 FROM prerequisite_courses_cache
    WHERE course_code = $1
) as is_prerequisite;
