-- catalog database: the current semester (:'semester', active) and the next
-- one (:'next_semester', planned), this semester's course offerings, the open
-- periods, and the next semester's prerequisite enrollment window.
--
-- Everything is relative to the seed day: periods are open today, and the
-- hard deadline is a full term away.
--
-- Instructor ids come from _staff — staff.staff is another database now.
\i /seed/sql/_staging-for-catalog.sql

-- ============================================================
-- Semesters
-- ============================================================
INSERT INTO course_catalog.semesters (name, status, hard_deadline, activated_at)
VALUES (:'semester', 'active', NOW() + INTERVAL '120 days', NOW())
ON CONFLICT (name) DO NOTHING;

-- Planned, so an admin can still add offerings to it from the UI: catalog
-- freezes a semester's course list once it is active.
INSERT INTO course_catalog.semesters (name, status, hard_deadline)
VALUES (:'next_semester', 'planned', NOW() + INTERVAL '300 days')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- This semester's offerings (Bilgisayar Mühendisliği)
-- ============================================================
-- Each teacher's own department. The prerequisite snapshot is the catalog's,
-- as an offering opened through the API would carry it.
INSERT INTO course_catalog.semester_courses
  (semester, course_code, department, credits, class_level, instructor_id, instructor_fullname,
   classroom_location, max_capacity, prerequisites, assessment_schema)
SELECT :'semester', c.course_code, c.department, c.credits, c.class_level, s.id,
       s.first_name || ' ' || s.last_name,
       'A Blok ' || (300 + (row_number() OVER (ORDER BY c.course_code)))::text,
       40,
       c.prerequisites,
       '[{"slug":"midterm","name":"Vize","weight":40},{"slug":"final","name":"Final","weight":60}]'::jsonb
FROM course_catalog.course_catalog c
JOIN (VALUES
   ('CENG101', 'ahmet.yilmaz@uni.edu.tr'),
   ('CENG102', 'ahmet.yilmaz@uni.edu.tr'),
   ('CENG201', 'elif.aydin@uni.edu.tr'),
   ('CENG202', 'elif.aydin@uni.edu.tr'),
   ('CENG301', 'mehmet.kaya@uni.edu.tr'),
   ('CENG350', 'mehmet.kaya@uni.edu.tr')
 ) AS m(course_code, email) ON m.course_code = c.course_code
JOIN _staff s ON s.email = m.email
ON CONFLICT (semester, course_code, department) DO NOTHING;

-- ============================================================
-- Periods — open today for every consuming service
-- ============================================================
-- Catalog holds one row per consuming service (source of truth); the
-- services read local projections, normally filled by catalog's period
-- events. The seed writes those projections directly in the per-service
-- files, the same shortcut it takes for the view tables.
--
-- Opened five weeks back rather than today: attendance has four weeks of
-- history (04-attendance.sql) that should fall inside its own period.
INSERT INTO course_catalog.academic_periods (semester, period_start, period_end, is_active, period_type)
SELECT :'semester', NOW() - INTERVAL '35 days', NOW() + INTERVAL '45 days', true, t.period_type
FROM (VALUES ('catalog'), ('enrollment'), ('grading'), ('attendance')) AS t(period_type)
ON CONFLICT (semester, period_type) DO NOTHING;

-- ============================================================
-- Prerequisite scenario — next semester's CENG201 requires CENG102
-- ============================================================
-- A separate semester so nobody's approved program for this one blocks a
-- fresh submission (programs are per-semester exclusive). The enrollment
-- period check reads enrollment.academic_periods, a projection of catalog's
-- enrollment-typed row.
--
-- Expected behaviour when a student POSTs a program for :'next_semester'
-- containing its CENG201 offering:
--   accept  → Zeynep, Emir, Eylül, Baran (passed CENG102, class_level ≥ 2)
--   reject  → Deniz, Selin  (class_level 2, but never passed CENG102 →
--             ErrPrerequisitesNotMet)
--   reject  → Kerem, Naz    (class_level 1 < 2 → ErrInvalidClassLevel, a
--             different, earlier guard — not the prerequisite check)
INSERT INTO course_catalog.academic_periods (semester, period_start, period_end, is_active, period_type)
SELECT :'next_semester', NOW() - INTERVAL '1 day', NOW() + INTERVAL '45 days', true, t.period_type
FROM (VALUES ('catalog'), ('enrollment'), ('grading'), ('attendance')) AS t(period_type)
ON CONFLICT (semester, period_type) DO NOTHING;

-- The snapshot the enrollment check walks (semester_courses.prerequisites,
-- keyed by course_code) is the catalog course's own list.
INSERT INTO course_catalog.semester_courses
  (semester, course_code, department, credits, class_level, instructor_id, instructor_fullname,
   classroom_location, max_capacity, prerequisites, assessment_schema)
SELECT :'next_semester', c.course_code, c.department, c.credits, c.class_level, s.id,
       s.first_name || ' ' || s.last_name, 'A Blok 401', 40,
       c.prerequisites,
       '[{"slug":"midterm","name":"Vize","weight":40},{"slug":"final","name":"Final","weight":60}]'::jsonb
FROM course_catalog.course_catalog c
JOIN _staff s ON s.email = 'elif.aydin@uni.edu.tr'
WHERE c.course_code = 'CENG201'
ON CONFLICT (semester, course_code, department) DO NOTHING;
