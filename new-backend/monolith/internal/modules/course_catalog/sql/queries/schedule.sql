-- name: CreateScheduleSession :one
INSERT INTO course_catalog.course_schedule_sessions (semester_course_id, day_of_week, slot_number, session_type)
VALUES ($1, $2, $3, $4)
RETURNING id, semester_course_id, day_of_week, slot_number, session_type, created_at;

-- name: BatchCreateScheduleSessions :batchexec
INSERT INTO course_catalog.course_schedule_sessions (semester_course_id, day_of_week, slot_number, session_type)
VALUES ($1, $2, $3, $4);

-- name: GetScheduleSessionsByCourseID :many
SELECT id, semester_course_id, day_of_week, slot_number, session_type, created_at
FROM course_catalog.course_schedule_sessions
WHERE semester_course_id = $1
ORDER BY session_type, day_of_week, slot_number;

-- name: GetScheduleSessionsByMultipleCourseIDs :many
SELECT id, semester_course_id, day_of_week, slot_number, session_type, created_at
FROM course_catalog.course_schedule_sessions
WHERE semester_course_id = ANY(sqlc.arg('course_ids')::UUID[])
ORDER BY semester_course_id, session_type, day_of_week, slot_number;

-- name: DeleteScheduleSessionsByCourseID :exec
DELETE FROM course_catalog.course_schedule_sessions
WHERE semester_course_id = $1;

-- name: CheckInstructorScheduleConflict :many
SELECT sc.course_code, sc.id, cc.department, css.day_of_week, css.slot_number
FROM course_catalog.course_schedule_sessions css
JOIN course_catalog.semester_courses sc ON css.semester_course_id = sc.id
JOIN course_catalog.course_catalog cc ON cc.course_code = sc.course_code
WHERE (css.day_of_week, css.slot_number) IN (
    SELECT unnest(sqlc.arg('days')::course_catalog.day_of_week_enum[]), unnest(sqlc.arg('slots')::SMALLINT[])
)
  AND sc.semester = sqlc.arg('semester')
  AND sc.instructor_id = sqlc.arg('instructor_id')
  AND sc.id != sqlc.arg('exclude_course_id')
ORDER BY css.day_of_week, css.slot_number;
