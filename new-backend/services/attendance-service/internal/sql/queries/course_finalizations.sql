-- name: InsertCourseFinalization :execrows
-- Zero rows means the course was already finalized; the caller rolls back.
INSERT INTO attendance.course_finalizations (
    course_id, semester, finalized_by, finalized_at, result
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (course_id) DO NOTHING;

-- name: GetCourseFinalization :one
SELECT course_id, semester, finalized_by, finalized_at, result
FROM attendance.course_finalizations
WHERE course_id = $1;
