-- name: GetStudentByID :one
SELECT id, student_number, email, first_name, last_name, department, class_level, advisor_id, status, is_active, synced_at
FROM students_cache
WHERE id = $1
LIMIT 1;

-- name: UpsertStudent :one
INSERT INTO students_cache (
    id, student_number, email, first_name, last_name, department, class_level, advisor_id, status, is_active, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
)
ON CONFLICT (id) DO UPDATE SET
    student_number = EXCLUDED.student_number,
    email = EXCLUDED.email,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    department = EXCLUDED.department,
    class_level = EXCLUDED.class_level,
    advisor_id = EXCLUDED.advisor_id,
    status = EXCLUDED.status,
    is_active = EXCLUDED.is_active,
    synced_at = NOW()
RETURNING id, student_number, email, first_name, last_name, department, class_level, advisor_id, status, is_active, synced_at;

-- name: DeactivateStudent :exec
UPDATE students_cache
SET is_active = false, synced_at = NOW()
WHERE id = $1;

-- name: GetStudentsByAdvisorID :many
SELECT id, student_number, email, first_name, last_name, department, class_level, advisor_id, status, is_active, synced_at
FROM students_cache
WHERE advisor_id = $1 AND is_active = true
ORDER BY student_number;
