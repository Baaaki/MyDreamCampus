-- catalog database: the current semester (:'semester', active) and the next
-- one (:'next_semester', planned), this semester's course offerings, the open
-- periods, the next semester's prerequisite enrollment window, and every
-- offering's weekly schedule.
--
-- Everything is relative to the seed day, and it has to hold for six months:
-- the hourly rewind always returns to this state, so its dates never move on
-- their own. Every period stays open 180 days; the active semester's hard
-- deadline falls just after (catalog refuses a period ending past it, and
-- completes the semester once it passes), the next one's a term later.
-- DEPLOY.md says how to refresh the state when the six months run out.
--
-- Instructor ids come from _staff — staff.staff is another database now.
\i /seed/sql/_staging-for-catalog.sql

-- ============================================================
-- Semesters
-- ============================================================
INSERT INTO course_catalog.semesters (name, status, hard_deadline, activated_at)
VALUES (:'semester', 'active', NOW() + INTERVAL '200 days', NOW())
ON CONFLICT (name) DO NOTHING;

-- Planned, so an admin can still add offerings to it from the UI: catalog
-- freezes a semester's course list once it is active.
INSERT INTO course_catalog.semesters (name, status, hard_deadline)
VALUES (:'next_semester', 'planned', NOW() + INTERVAL '380 days')
ON CONFLICT (name) DO NOTHING;

-- ============================================================
-- This semester's offerings (Bilgisayar Mühendisliği)
-- ============================================================
-- Each teacher's own department. The prerequisite snapshot is the catalog's,
-- as an offering opened through the API would carry it.
--
-- CENG301 is ahmet.yilmaz's so the demo teacher and the demo student
-- (zeynep.sahin, third year: CENG301 + CENG350) share a course: his QR is one
-- she can scan, and the final he enters is one she sees.
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
   ('CENG301', 'ahmet.yilmaz@uni.edu.tr'),
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
SELECT :'semester', NOW() - INTERVAL '35 days', NOW() + INTERVAL '180 days', true, t.period_type
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
SELECT :'next_semester', NOW() - INTERVAL '1 day', NOW() + INTERVAL '180 days', true, t.period_type
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

-- ============================================================
-- Weekly schedule — every offering of both semesters
-- ============================================================
-- What the offering form would have saved: one slot per catalog hour, theory
-- and lab counted apart (the API refuses any other count). Slot 5 is the
-- lunch break. No two offerings of a semester share a slot, so a teacher's
-- courses never collide (trg_instructor_schedule_conflict would refuse it) and
-- neither does any pair a student takes together (CENG301+CENG350,
-- CENG102+CENG202). Next semester's CENG201 keeps this term's hours.
--
-- NOT EXISTS rather than ON CONFLICT: the conflict trigger fires before the
-- arbiter, and on a re-run it would take an offering's own rows for a clash.
INSERT INTO course_catalog.course_schedule_sessions (semester_course_id, day_of_week, slot_number, session_type)
SELECT sc.id, s.day::course_catalog.day_of_week_enum, slot.n,
       s.session_type::course_catalog.schedule_session_type_enum
FROM course_catalog.semester_courses sc
JOIN (VALUES
   ('CENG101', 'monday',    'theory', ARRAY[1, 2, 3]),
   ('CENG101', 'wednesday', 'lab',    ARRAY[6, 7]),
   ('CENG102', 'tuesday',   'theory', ARRAY[1, 2, 3]),
   ('CENG102', 'thursday',  'lab',    ARRAY[6, 7]),
   ('CENG201', 'monday',    'theory', ARRAY[6, 7, 8]),
   ('CENG202', 'tuesday',   'theory', ARRAY[6, 7]),
   ('CENG202', 'friday',    'lab',    ARRAY[1, 2]),
   ('CENG301', 'wednesday', 'theory', ARRAY[1, 2, 3]),
   ('CENG350', 'thursday',  'theory', ARRAY[1, 2, 3])
 ) AS s(course_code, day, session_type, slots) ON s.course_code = sc.course_code
CROSS JOIN LATERAL unnest(s.slots) AS slot(n)
WHERE sc.semester IN (:'semester', :'next_semester')
  AND NOT EXISTS (SELECT 1 FROM course_catalog.course_schedule_sessions x
                  WHERE x.semester_course_id = sc.id);
