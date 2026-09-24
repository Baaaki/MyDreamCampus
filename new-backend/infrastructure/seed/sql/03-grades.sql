-- grades database: read-model views, this semester's registrations with the
-- midterm entered and the final still open, the cohort's transcripts, the
-- grading periods and the hidden prerequisite index.
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
-- Every semester, not just this one: approving a next-semester program emits
-- enrollment.program.approved carrying that offering's id, and the
-- registration this consumer writes has an FK onto courses_view. Without the
-- row that event dead-letters — the very flow the prerequisite scenario is
-- built to exercise.
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- This semester: midterm in and locked, final not yet entered
-- ============================================================
-- The teacher's gradebook opens on an empty final column; the student sees a
-- midterm and a pending final. zeynep.sahin's two midterms are the old mock
-- report card's, everyone else's a stable hash.
INSERT INTO grades.student_course_registrations (student_id, course_id, semester)
SELECT p.student_id, p.course_id, :'semester'
FROM _programs p
WHERE p.status = 'approved'
ON CONFLICT (student_id, course_id) DO NOTHING;

INSERT INTO grades.student_assessment_scores (registration_id, slug, score, graded_by, graded_at, is_locked)
SELECT r.id, 'midterm',
       COALESCE(
         CASE WHEN p.student_number = '2021510001' AND p.course_code = 'CENG301' THEN 78
              WHEN p.student_number = '2021510001' AND p.course_code = 'CENG350' THEN 85 END,
         45 + (abs(hashtext(r.id::text || 'midterm')) % 53)
       )::numeric(5,2),
       cv.instructor_id, NOW() - INTERVAL '5 days', true
FROM grades.student_course_registrations r
JOIN grades.courses_view cv ON cv.id = r.course_id
JOIN _programs p ON p.student_id = r.student_id AND p.course_id = r.course_id
WHERE r.semester = :'semester'
ON CONFLICT (registration_id, slug) DO NOTHING;

-- ============================================================
-- Transcripts
-- ============================================================
-- Finalised courses are snapshots with no FK: the offering of a past
-- semester never existed in this system, so course_id is the catalog id.
-- class_statistics is what finalisation would have frozen for the section.
INSERT INTO grades.student_completed_courses
  (student_id, student_number, student_first_name, student_last_name, student_department,
   course_id, course_code, course_name, credits, semester,
   instructor_id, instructor_name, assessment_scores, weighted_average, grade_point,
   grading_type, grading_config, class_statistics, is_attendance_failed, finalized_at, finalized_by)
SELECT cc.student_id, cc.student_number, cc.first_name, cc.last_name, cc.department,
       cc.course_id, cc.course_code, cc.course_name, cc.credits, cc.semester,
       sc.instructor_id, sc.instructor_fullname,
       jsonb_build_object('midterm', cc.midterm, 'final', cc.final),
       cc.weighted_average,
       (SELECT g.grade_point FROM _grade_scale g WHERE cc.weighted_average >= g.min_average
        ORDER BY g.min_average DESC LIMIT 1)::grades.grade_point_enum,
       'absolute', '{}'::jsonb,
       jsonb_build_object(
         'total_students', count(*) OVER w,
         'mean',   round(avg(cc.weighted_average) OVER w, 2),
         'stddev', round(coalesce(stddev_pop(cc.weighted_average) OVER w, 0), 2),
         'min',    min(cc.weighted_average) OVER w,
         'max',    max(cc.weighted_average) OVER w),
       false, cc.finalized_on + TIME '17:00', sc.instructor_id
FROM _completed cc
-- The same teacher who teaches the course this semester taught it then.
JOIN _semester_courses sc ON sc.course_code = cc.course_code AND sc.semester = :'semester'
WINDOW w AS (PARTITION BY cc.course_code, cc.semester)
ON CONFLICT (student_id, course_id) DO NOTHING;

-- ============================================================
-- Grading periods + hidden prerequisite index
-- ============================================================
INSERT INTO grades.academic_periods (id, semester, period_start, period_end, is_active)
SELECT id, semester, period_start, period_end, COALESCE(is_active, true)
FROM _periods
WHERE period_type = 'grading'
ON CONFLICT (semester) DO NOTHING;

-- grades' "gizli tablo": which courses are some offering's prerequisite.
-- Normally filled by course.semester.created events; the seed opened its
-- offerings in SQL, so it fills this from the same snapshots.
INSERT INTO grades.prerequisite_courses_view (course_code, course_id)
SELECT DISTINCT p ->> 'course_code', (p ->> 'id')::uuid
FROM _semester_courses sc
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(sc.prerequisites, '[]'::jsonb)) AS p
ON CONFLICT (course_code, course_id) DO NOTHING;
