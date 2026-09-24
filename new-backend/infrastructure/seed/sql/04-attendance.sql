-- attendance database: read-model views, the periods, four past weeks of
-- this semester's sessions and who attended them. Week 5 is left open: that
-- is the session a teacher starts from the UI today.
\i /seed/sql/_staging-for-services.sql

INSERT INTO attendance.students_view (id, student_number, first_name, last_name, email, department, is_active)
SELECT id, student_number, first_name, last_name, email, department, is_active
FROM _students
ON CONFLICT (id) DO NOTHING;

INSERT INTO attendance.courses_view
  (id, course_code, course_name, credits, semester, department, instructor_id, instructor_fullname, total_weeks, has_lab)
SELECT sc.id, sc.course_code, c.name, sc.credits, sc.semester, c.department,
       sc.instructor_id, sc.instructor_fullname, 14, c.lab_hours > 0
FROM _semester_courses sc
JOIN _courses c ON c.course_code = sc.course_code
-- Every semester, not just this one: approving a next-semester program emits
-- enrollment.program.approved carrying that offering's id, and
-- enrollments_view has an FK onto this table.
ON CONFLICT (id) DO NOTHING;

INSERT INTO attendance.enrollments_view (student_id, course_id, semester)
SELECT p.student_id, p.course_id, :'semester'
FROM _programs p
WHERE p.status = 'approved'
ON CONFLICT (student_id, course_id) DO NOTHING;

INSERT INTO attendance.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE period_type = 'attendance'
ON CONFLICT (semester) DO NOTHING;

-- ============================================================
-- Weeks 1-4: closed theory sessions, one a week per course
-- ============================================================
-- Spread over the weekdays by course so a week does not read as six classes
-- on one morning. Four, not three: attendance requires 10 of 14 sessions
-- pro rata, and at three held sessions a single absence already reads as
-- failing the course.
INSERT INTO attendance.attendance_sessions
  (course_id, instructor_id, semester, week_number, session_date, session_type,
   qr_secret, started_at, expires_at, is_active)
SELECT cv.id, cv.instructor_id, :'semester', w.week, d.day, 'theory',
       substr(md5(random()::text), 1, 32), d.day + TIME '09:00', d.day + TIME '09:15', false
FROM attendance.courses_view cv
CROSS JOIN (VALUES (1), (2), (3), (4)) AS w(week)
CROSS JOIN LATERAL (
    SELECT (date_trunc('week', CURRENT_DATE)::date - (5 - w.week) * 7
            + (abs(hashtext(cv.course_code)) % 5)) AS day
) d
WHERE cv.semester = :'semester'
ON CONFLICT (course_id, week_number, session_type) DO NOTHING;

-- Present = a record. Roughly one student in eight misses a given week.
-- zeynep.sahin misses exactly week 2 of CENG301: her history shows an absence
-- and she is still above the attendance minimum.
INSERT INTO attendance.attendance_records
  (session_id, student_id, course_id, semester, week_number, session_type,
   marked_via, scanned_at)
SELECT s.id, e.student_id, s.course_id, s.semester, s.week_number, s.session_type,
       'qr_scan', s.started_at + (abs(hashtext(e.student_id::text || s.id::text)) % 600) * INTERVAL '1 second'
FROM attendance.attendance_sessions s
JOIN attendance.enrollments_view e ON e.course_id = s.course_id
JOIN attendance.students_view sv ON sv.id = e.student_id
JOIN attendance.courses_view cv ON cv.id = s.course_id
WHERE s.semester = :'semester'
  AND CASE WHEN sv.student_number = '2021510001'
           THEN NOT (cv.course_code = 'CENG301' AND s.week_number = 2)
           ELSE abs(hashtext(e.student_id::text || s.week_number::text || cv.course_code)) % 8 <> 0
      END
ON CONFLICT (session_id, student_id) DO NOTHING;
