-- name: UpsertCourseCache :one
INSERT INTO courses_cache (
    id, course_code, course_name, credits, semester, department,
    instructor_id, instructor_fullname, assessment_schema, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()
)
ON CONFLICT (id) DO UPDATE SET
    course_code = EXCLUDED.course_code,
    course_name = EXCLUDED.course_name,
    credits = EXCLUDED.credits,
    semester = EXCLUDED.semester,
    department = EXCLUDED.department,
    instructor_id = EXCLUDED.instructor_id,
    instructor_fullname = EXCLUDED.instructor_fullname,
    assessment_schema = EXCLUDED.assessment_schema,
    synced_at = NOW()
RETURNING *;

-- name: GetCourseCacheByID :one
SELECT * FROM courses_cache
WHERE id = $1;

