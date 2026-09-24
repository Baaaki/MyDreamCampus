-- enrollment database: this semester's programs (approved, and three waiting
-- on their advisor), the enrollment windows, and the passed-prerequisite
-- projection the next semester's scenario leans on.
\i /seed/sql/_staging-for-services.sql

-- course_id is the OFFERING's id, the one available-courses hands a student:
-- approving a program publishes these ids, and grades and attendance key
-- their registrations on them.
INSERT INTO enrollment.enrollment_programs (student_id, semester, status, created_at)
SELECT DISTINCT p.student_id, :'semester', p.status::enrollment.enrollment_status_enum,
       NOW() - CASE p.status WHEN 'pending' THEN INTERVAL '2 days' ELSE INTERVAL '25 days' END
FROM _programs p
ON CONFLICT (student_id, semester) DO NOTHING;

INSERT INTO enrollment.enrollment_program_courses (program_id, course_id, course_code, course_name, credits)
SELECT ep.id, p.course_id, p.course_code, p.course_name, p.credits
FROM _programs p
JOIN enrollment.enrollment_programs ep ON ep.student_id = p.student_id AND ep.semester = :'semester'
ON CONFLICT (program_id, course_id) DO NOTHING;

-- The enrollment period check reads this table, not catalog's.
INSERT INTO enrollment.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE period_type = 'enrollment'
ON CONFLICT (semester) DO NOTHING;

-- In production these rows arrive via grades' grade.student.prerequisite.passed
-- event on finalize; the seed fills the projection from the same transcripts
-- 03-grades.sql writes. Only CENG102 is anyone's prerequisite in this
-- department — Deniz and Selin are the two class-2 students without it.
INSERT INTO enrollment.student_passed_prerequisites (student_id, course_id, course_code, semester, grade_point)
SELECT c.student_id, c.course_id, c.course_code, c.semester,
       (SELECT g.grade_point FROM _grade_scale g WHERE c.weighted_average >= g.min_average
        ORDER BY g.min_average DESC LIMIT 1)
FROM _completed c
WHERE c.course_code = 'CENG102'
ON CONFLICT (student_id, course_code) DO NOTHING;
