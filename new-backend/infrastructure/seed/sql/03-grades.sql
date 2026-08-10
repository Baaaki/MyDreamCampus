-- grades database: read-model views, registrations, midterm/final scores, the
-- Bahar grading period and the hidden prerequisite index.
--
-- The view tables are normally filled by events. Filling them here too is safe
-- and idempotent — ON CONFLICT DO NOTHING lets whichever path arrives first win.
\i /seed/sql/_staging-for-services.sql

INSERT INTO grades.students_view (id, student_number, first_name, last_name, email, department, class_level, is_active)
SELECT id, student_number, first_name, last_name, email, department, class_level, is_active
FROM _students
ON CONFLICT (id) DO NOTHING;

INSERT INTO grades.courses_view
  (id, course_code, course_name, credits, semester, department, instructor_id, instructor_fullname, assessment_schema)
SELECT sc.id, sc.course_code, c.name, sc.credits, sc.semester, c.department,
       sc.instructor_id, sc.instructor_fullname, sc.assessment_schema
FROM _semester_courses sc
JOIN _courses c ON c.course_code = sc.course_code
-- Every semester, not just Güz: approving a Bahar program emits
-- enrollment.program.approved carrying the Bahar offering's course id, and
-- the registration this consumer writes has an FK onto courses_view. Without
-- the Bahar row that event dead-letters — which is exactly the flow the
-- prerequisite demo below is built to exercise. Nothing further down keys off
-- this view without scoping itself to Güz first.
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Registrations + midterm/final scores for those courses
-- ============================================================
INSERT INTO grades.student_course_registrations (student_id, course_id, semester)
SELECT sv.id, cv.id, '2025-2026 Güz'
FROM grades.students_view sv
JOIN grades.courses_view cv
  ON cv.semester = '2025-2026 Güz' AND cv.course_code IN ('CENG101','CENG102','CENG201')
ON CONFLICT (student_id, course_id) DO NOTHING;

INSERT INTO grades.student_assessment_scores (registration_id, slug, score, graded_by)
SELECT r.id, v.slug, v.score, cv.instructor_id
FROM grades.student_course_registrations r
JOIN grades.courses_view cv ON cv.id = r.course_id
CROSS JOIN LATERAL (VALUES
   ('midterm', (55 + floor(random() * 40))::numeric(5,2)),
   ('final',   (50 + floor(random() * 45))::numeric(5,2))
 ) AS v(slug, score)
ON CONFLICT (registration_id, slug) DO NOTHING;

-- ============================================================
-- Prerequisite demo — grading period + hidden prerequisite index
-- ============================================================
INSERT INTO grades.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE semester = '2025-2026 Bahar' AND period_type = 'grading'
ON CONFLICT (semester) DO NOTHING;

-- grades' "gizli tablo": CENG102 is now a prerequisite of something. Normally
-- filled by course.semester.created events; filled directly here, same as the
-- read-model views above.
INSERT INTO grades.prerequisite_courses_view (course_code, course_id)
SELECT 'CENG102', id FROM _courses WHERE course_code = 'CENG102'
ON CONFLICT (course_code, course_id) DO NOTHING;
