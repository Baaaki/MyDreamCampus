-- name: CreateAttendanceRecordQR :exec
INSERT INTO attendance.attendance_records (
    session_id, student_id, course_id, semester, week_number,
    marked_via, scanned_at, qr_timestamp, session_type
) VALUES (
    $1, $2, $3, $4, $5, 'qr_scan', $6, $7, $8
) ON CONFLICT (session_id, student_id) DO NOTHING;

-- name: BatchCreateAttendanceRecordsQR :exec
INSERT INTO attendance.attendance_records (
    session_id, student_id, course_id, semester, week_number,
    marked_via, scanned_at, qr_timestamp, session_type
)
SELECT
    session_id, student_id, course_id, semester, week_number,
    'qr_scan', scanned_at, qr_timestamp, session_type
FROM (
    SELECT
        unnest(@session_ids::uuid[])            AS session_id,
        unnest(@student_ids::uuid[])            AS student_id,
        unnest(@course_ids::uuid[])             AS course_id,
        unnest(@semesters::text[])              AS semester,
        unnest(@week_numbers::smallint[])       AS week_number,
        unnest(@scanned_ats::timestamp[])       AS scanned_at,
        unnest(@qr_timestamps::bigint[])        AS qr_timestamp,
        unnest(@session_types::session_type_enum[]) AS session_type
) AS input
ON CONFLICT (session_id, student_id) DO NOTHING;

-- name: CreateAttendanceRecordManual :one
INSERT INTO attendance.attendance_records (
    session_id, student_id, course_id, semester, week_number,
    marked_via, manually_marked_by, manually_marked_at, manual_note, session_type
) VALUES (
    $1, $2, $3, $4, $5, 'manual', $6, NOW(), $7, $8
) ON CONFLICT (session_id, student_id)
DO UPDATE SET
    marked_via = EXCLUDED.marked_via,
    manually_marked_by = EXCLUDED.manually_marked_by,
    manually_marked_at = NOW(),
    manual_note = EXCLUDED.manual_note
RETURNING *;

-- name: GetMarkedStudentsBySession :many
SELECT student_id
FROM attendance.attendance_records
WHERE session_id = $1;

-- name: GetSessionAttendanceCount :one
SELECT COUNT(*) as present_count
FROM attendance.attendance_records
WHERE session_id = $1;

-- name: GetStudentAttendanceByCourse :many
SELECT
    ar.week_number,
    ats.session_date,
    ar.marked_via,
    ar.manual_note,
    ar.scanned_at,
    ar.manually_marked_at,
    ar.session_type
FROM attendance.attendance_records ar
JOIN attendance.attendance_sessions ats ON ar.session_id = ats.id
WHERE ar.student_id = $1 AND ar.course_id = $2 AND ar.semester = $3
ORDER BY ar.week_number ASC, ar.session_type ASC;

-- name: GetCourseAttendanceStats :many
SELECT
    s.id as student_id,
    s.student_number,
    s.first_name,
    s.last_name,
    COUNT(ar.id) as present_count
FROM attendance.enrollments_view e
JOIN attendance.students_view s ON e.student_id = s.id
LEFT JOIN attendance.attendance_records ar ON ar.student_id = s.id AND ar.course_id = e.course_id AND ar.semester = e.semester
WHERE e.course_id = $1 AND e.semester = $2
GROUP BY s.id, s.student_number, s.first_name, s.last_name
ORDER BY s.student_number;

-- name: GetCourseAttendanceStatsByType :many
SELECT
    s.id as student_id,
    s.student_number,
    s.first_name,
    s.last_name,
    COUNT(ar.id) as present_count
FROM attendance.enrollments_view e
JOIN attendance.students_view s ON e.student_id = s.id
LEFT JOIN attendance.attendance_records ar ON ar.student_id = s.id AND ar.course_id = e.course_id AND ar.semester = e.semester AND ar.session_type = $3
WHERE e.course_id = $1 AND e.semester = $2
GROUP BY s.id, s.student_number, s.first_name, s.last_name
ORDER BY s.student_number;

-- name: GetFailingStudentsByCourse :many
SELECT
    s.id as student_id,
    s.student_number,
    s.first_name,
    s.last_name,
    s.email,
    COUNT(ar.id) as present_count,
    (sqlc.arg(total_sessions)::bigint - COUNT(ar.id))::bigint as absent_count
FROM attendance.enrollments_view e
JOIN attendance.students_view s ON e.student_id = s.id
LEFT JOIN attendance.attendance_records ar ON ar.student_id = s.id AND ar.course_id = e.course_id AND ar.semester = e.semester
WHERE e.course_id = $1 AND e.semester = $2
GROUP BY s.id, s.student_number, s.first_name, s.last_name, s.email
HAVING (sqlc.arg(total_sessions)::bigint - COUNT(ar.id)) > sqlc.arg(max_allowed_absences)::bigint;

-- name: GetFailingStudentsByCourseByType :many
SELECT
    s.id as student_id,
    s.student_number,
    s.first_name,
    s.last_name,
    s.email,
    COUNT(ar.id) as present_count,
    (sqlc.arg(total_sessions)::bigint - COUNT(ar.id))::bigint as absent_count
FROM attendance.enrollments_view e
JOIN attendance.students_view s ON e.student_id = s.id
LEFT JOIN attendance.attendance_records ar ON ar.student_id = s.id AND ar.course_id = e.course_id AND ar.semester = e.semester AND ar.session_type = $3
WHERE e.course_id = $1 AND e.semester = $2
GROUP BY s.id, s.student_number, s.first_name, s.last_name, s.email
HAVING COUNT(ar.id) < sqlc.arg(min_required_attendance)::bigint;

-- name: GetAttendanceRecordsBySession :many
SELECT
    ar.id,
    ar.session_id,
    ar.student_id,
    ar.course_id,
    ar.semester,
    ar.week_number,
    ar.marked_via,
    ar.scanned_at,
    ar.qr_timestamp,
    ar.manually_marked_by,
    ar.manually_marked_at as marked_at,
    ar.manual_note,
    ar.created_at,
    ar.session_type
FROM attendance.attendance_records ar
WHERE ar.session_id = $1
ORDER BY ar.created_at DESC;

-- name: GetTotalSessionsByCourse :one
SELECT COUNT(*) as total_sessions
FROM attendance.attendance_sessions
WHERE course_id = $1 AND semester = $2;

-- name: GetTotalSessionsByCourseAndType :one
SELECT COUNT(*) as total_sessions
FROM attendance.attendance_sessions
WHERE course_id = $1 AND semester = $2 AND session_type = $3;

-- name: GetStudentPresentCountByType :one
SELECT COUNT(*) as present_count
FROM attendance.attendance_records
WHERE student_id = $1 AND course_id = $2 AND semester = $3 AND session_type = $4;
