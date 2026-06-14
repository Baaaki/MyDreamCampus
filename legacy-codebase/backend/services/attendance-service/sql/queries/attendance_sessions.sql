-- name: CreateAttendanceSession :one
INSERT INTO attendance_sessions (
    course_id, instructor_id, semester, week_number, session_date,
    qr_secret, qr_rotation_interval, started_at, expires_at, is_active, session_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, $10
) RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM attendance_sessions
WHERE id = $1
LIMIT 1;

-- name: GetActiveSessionByID :one
SELECT * FROM attendance_sessions
WHERE id = $1 AND is_active = TRUE AND expires_at > NOW()
LIMIT 1;

-- name: CheckSessionExists :one
SELECT COUNT(*) as count
FROM attendance_sessions
WHERE course_id = $1 AND week_number = $2 AND session_type = $3;

-- name: GetSessionsByCourse :many
SELECT
    id,
    week_number,
    session_date,
    is_active,
    started_at,
    expires_at,
    session_type
FROM attendance_sessions
WHERE course_id = $1 AND semester = $2
ORDER BY week_number ASC, session_type ASC;

-- name: DeactivateSession :exec
UPDATE attendance_sessions
SET is_active = FALSE
WHERE id = $1;

-- name: GetExpiredSessions :many
SELECT * FROM attendance_sessions
WHERE is_active = TRUE AND expires_at < NOW();

-- name: GetSessionsByDateRange :many
SELECT
    s.id,
    s.course_id,
    s.instructor_id,
    s.semester,
    s.week_number,
    s.session_date,
    s.session_type,
    s.is_active,
    s.started_at,
    s.expires_at,
    c.course_code,
    c.course_name,
    (SELECT COUNT(*) FROM attendance_records ar WHERE ar.session_id = s.id) as present_count,
    (SELECT COUNT(*) FROM enrollments_cache ec WHERE ec.course_id = s.course_id AND ec.semester = s.semester) as enrolled_count
FROM attendance_sessions s
JOIN courses_cache c ON s.course_id = c.id
WHERE s.session_date >= @start_date AND s.session_date <= @end_date
ORDER BY s.session_date ASC, s.started_at ASC;
