-- Clear existing courses for this semester/dept to avoid clutter (optional, but good for testing)
-- DELETE FROM semester_courses_cache WHERE semester = '2026-2027-Fall' AND department = 'Bilgisayar Mühendisliği';

-- Year 1
WITH c1 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM101', 'Bilgisayar Mühendisliğine Giriş', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 3, 'mandatory', 1,
        '2026-2027-Fall', gen_random_uuid(), 'Dr. Ali Veli', 'D-101', 50, 0, '[]', NOW()
    ) RETURNING id
),
c2 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM103', 'Programlama I', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 4, 'mandatory', 1,
        '2026-2027-Fall', gen_random_uuid(), 'Dr. Ayşe Yılmaz', 'Lab-1', 40, 0, '[]', NOW()
    ) RETURNING id
),

-- Year 2
c3 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM201', 'Veri Yapıları', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 4, 'mandatory', 2,
        '2026-2027-Fall', gen_random_uuid(), 'Prof. Mehmet Öz', 'D-201', 60, 0, '[]', NOW()
    ) RETURNING id
),
c4 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM205', 'Sayısal Devreler', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 3, 'mandatory', 2,
        '2026-2027-Fall', gen_random_uuid(), 'Dr. Fatma Demir', 'Lab-2', 45, 0, '[]', NOW()
    ) RETURNING id
),

-- Year 3
c5 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM301', 'İşletim Sistemleri', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 4, 'mandatory', 3,
        '2026-2027-Fall', gen_random_uuid(), 'Doc. Caner Erkin', 'D-301', 55, 0, '[]', NOW()
    ) RETURNING id
),
c6 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM305', 'Algoritma Analizi', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 3, 'mandatory', 3,
        '2026-2027-Fall', gen_random_uuid(), 'Prof. Zeynep Kaya', 'D-302', 50, 0, '[]', NOW()
    ) RETURNING id
),

-- Year 4
c7 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM401', 'Bitirme Projesi I', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 5, 'mandatory', 4,
        '2026-2027-Fall', gen_random_uuid(), 'Dr. Ali Veli', 'Ofis', 100, 0, '[]', NOW()
    ) RETURNING id
),
c8 AS (
    INSERT INTO semester_courses_cache (
        id, course_code, course_name, faculty, department, credits, course_type, class_level,
        semester, instructor_id, instructor_fullname, classroom_location, max_capacity,
        current_enrollment, prerequisites, synced_at
    ) VALUES (
        gen_random_uuid(), 'BLM405', 'Ağ Güvenliği', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 3, 'elective', 4,
        '2026-2027-Fall', gen_random_uuid(), 'Doc. Burak Yılmaz', 'D-401', 30, 0, '[]', NOW()
    ) RETURNING id
)

-- Insert Sessions
INSERT INTO course_sessions_cache (id, course_id, day_of_week, slot_number, synced_at)
SELECT gen_random_uuid(), id, 'monday'::day_of_week_enum, 1, NOW() FROM c1 -- BLM101: Mon 09:00
UNION ALL
SELECT gen_random_uuid(), id, 'monday'::day_of_week_enum, 2, NOW() FROM c1 -- BLM101: Mon 10:00
UNION ALL
SELECT gen_random_uuid(), id, 'tuesday'::day_of_week_enum, 3, NOW() FROM c2 -- BLM103: Tue 11:00
UNION ALL
SELECT gen_random_uuid(), id, 'tuesday'::day_of_week_enum, 4, NOW() FROM c2 -- BLM103: Tue 12:00
UNION ALL
SELECT gen_random_uuid(), id, 'wednesday'::day_of_week_enum, 1, NOW() FROM c3 -- BLM201: Wed 09:00
UNION ALL
SELECT gen_random_uuid(), id, 'wednesday'::day_of_week_enum, 2, NOW() FROM c3 -- BLM201: Wed 10:00
UNION ALL
SELECT gen_random_uuid(), id, 'thursday'::day_of_week_enum, 5, NOW() FROM c4 -- BLM205: Thu 13:00
UNION ALL
SELECT gen_random_uuid(), id, 'thursday'::day_of_week_enum, 6, NOW() FROM c4 -- BLM205: Thu 14:00
UNION ALL
SELECT gen_random_uuid(), id, 'friday'::day_of_week_enum, 1, NOW() FROM c5 -- BLM301: Fri 09:00
UNION ALL
SELECT gen_random_uuid(), id, 'friday'::day_of_week_enum, 2, NOW() FROM c5 -- BLM301: Fri 10:00
UNION ALL
SELECT gen_random_uuid(), id, 'friday'::day_of_week_enum, 5, NOW() FROM c6 -- BLM305: Fri 13:00
UNION ALL
SELECT gen_random_uuid(), id, 'monday'::day_of_week_enum, 7, NOW() FROM c7 -- BLM401: Mon 15:00
UNION ALL
SELECT gen_random_uuid(), id, 'wednesday'::day_of_week_enum, 7, NOW() FROM c8; -- BLM405: Wed 15:00
