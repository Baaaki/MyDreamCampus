-- attendance database: read-model views, the active period, three weeks of
-- sessions and a present record for every enrolled student.
\i /seed/sql/_staging-for-services.sql

INSERT INTO attendance.students_view (id, student_number, first_name, last_name, email, department, is_active)
SELECT id, student_number, first_name, last_name, email, department, is_active
FROM _students
ON CONFLICT (id) DO NOTHING;

INSERT INTO attendance.courses_view
  (id, course_code, course_name, credits, semester, department, instructor_id, instructor_fullname, total_weeks, has_lab)
SELECT sc.id, sc.course_code, c.name, sc.credits, sc.semester, c.department,
       sc.instructor_id, sc.instructor_fullname, 14, false
FROM _semester_courses sc
JOIN _courses c ON c.course_code = sc.course_code
-- Every semester, not just Güz: approving a Bahar program emits
-- enrollment.program.approved carrying the Bahar offering's course id, and
-- enrollments_view has an FK onto this table. Without the Bahar row that
-- event dead-letters — the very flow the prerequisite demo sets up. The
-- inserts below all scope themselves to Güz, so nothing else changes.
ON CONFLICT (id) DO NOTHING;

INSERT INTO attendance.enrollments_view (student_id, course_id, semester)
SELECT sv.id, cv.id, '2025-2026 Güz'
FROM attendance.students_view sv
JOIN attendance.courses_view cv
  ON cv.semester = '2025-2026 Güz' AND cv.course_code IN ('CENG101','CENG102','CENG201')
ON CONFLICT (student_id, course_id) DO NOTHING;

INSERT INTO attendance.academic_periods (semester, period_start, period_end, is_active)
VALUES ('2025-2026 Güz', NOW() - INTERVAL '30 days', NOW() + INTERVAL '90 days', true)
ON CONFLICT (semester) DO NOTHING;

INSERT INTO attendance.attendance_sessions
  (course_id, instructor_id, semester, week_number, session_date, session_type, qr_secret, expires_at, is_active)
SELECT cv.id, cv.instructor_id, '2025-2026 Güz', w.week,
       CURRENT_DATE - ((4 - w.week) * 7), 'theory',
       substr(md5(random()::text), 1, 32), NOW(), false
FROM attendance.courses_view cv
CROSS JOIN (VALUES (1), (2), (3)) AS w(week)
WHERE cv.semester = '2025-2026 Güz' AND cv.course_code IN ('CENG101','CENG102','CENG201')
ON CONFLICT (course_id, week_number, session_type) DO NOTHING;

INSERT INTO attendance.attendance_records
  (session_id, student_id, course_id, semester, week_number, session_type, marked_via, manually_marked_at)
SELECT s.id, e.student_id, s.course_id, s.semester, s.week_number, s.session_type, 'admin', NOW()
FROM attendance.attendance_sessions s
JOIN attendance.enrollments_view e ON e.course_id = s.course_id
WHERE s.semester = '2025-2026 Güz'
ON CONFLICT (session_id, student_id) DO NOTHING;

-- Prerequisite demo: the Bahar attendance window, projected from catalog.
INSERT INTO attendance.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE semester = '2025-2026 Bahar' AND period_type = 'attendance'
ON CONFLICT (semester) DO NOTHING;
