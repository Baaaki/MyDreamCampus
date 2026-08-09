-- enrollment database: an approved program per student, the Bahar enrollment
-- window, and the passed-prerequisite projection the demo leans on.
\i /seed/sql/_staging-for-services.sql

INSERT INTO enrollment.enrollment_programs (student_id, semester, status)
SELECT id, '2025-2026 Güz', 'approved' FROM _students
ON CONFLICT (student_id, semester) DO NOTHING;

INSERT INTO enrollment.enrollment_program_courses (program_id, course_id, course_code, course_name, credits)
SELECT p.id, c.id, c.course_code, c.name, c.credits
FROM enrollment.enrollment_programs p
JOIN _courses c ON c.course_code IN ('CENG101','CENG102','CENG201')
WHERE p.semester = '2025-2026 Güz'
ON CONFLICT (program_id, course_id) DO NOTHING;

-- Prerequisite demo: the Bahar enrollment window, projected from catalog. The
-- enrollment period check reads this table, not catalog's.
INSERT INTO enrollment.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE semester = '2025-2026 Bahar' AND period_type = 'enrollment'
ON CONFLICT (semester) DO NOTHING;

-- In production these rows arrive via grades' grade.student.prerequisite.passed
-- event on finalize; the seed fills the projection directly (same philosophy as
-- every view above). Only the four upper-class students who "passed CENG102".
INSERT INTO enrollment.student_passed_prerequisites (student_id, course_id, course_code, semester, grade_point)
SELECT st.id,
       (SELECT id FROM _courses WHERE course_code = 'CENG102'),
       'CENG102', '2024-2025 Bahar', v.grade_point
FROM _students st
JOIN (VALUES
   ('2021510001', '3.50'),  -- Zeynep
   ('2021510002', '3.00'),  -- Emir
   ('2022510010', '2.50'),  -- Elif
   ('2022510011', '2.00')   -- Baran
 ) AS v(student_number, grade_point) ON v.student_number = st.student_number
ON CONFLICT (student_id, course_code) DO NOTHING;
