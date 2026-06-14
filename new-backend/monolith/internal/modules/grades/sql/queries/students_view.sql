-- name: UpsertStudentCache :one
INSERT INTO grades.students_view (
    id, student_number, first_name, last_name, email, department, class_level, is_active, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
)
ON CONFLICT (id) DO UPDATE SET
    student_number = EXCLUDED.student_number,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email = EXCLUDED.email,
    department = EXCLUDED.department,
    class_level = EXCLUDED.class_level,
    is_active = EXCLUDED.is_active,
    synced_at = NOW()
RETURNING *;

-- name: GetStudentCacheByID :one
SELECT * FROM grades.students_view
WHERE id = $1;

-- name: DeactivateStudentCache :exec
UPDATE grades.students_view
SET is_active = false, synced_at = NOW()
WHERE id = $1;
