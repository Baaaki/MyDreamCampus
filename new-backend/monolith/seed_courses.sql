-- seed_courses.sql
-- Seed script for Course Catalog

INSERT INTO course_catalog.course_catalog (
    id, course_code, name, faculty, department, class_level, semester,
    credits, ects, theoretical_hours, lab_hours, course_type,
    course_category, education_level, teaching_type, language, status
) VALUES
(gen_random_uuid(), 'BB101', 'Bilgisayar Bilimine Giriş', 'Fen Fakültesi', 'Bilgisayar Bilimi', 1, 1, 3, 5, 3, 0, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BB103', 'Programlama Temelleri', 'Fen Fakültesi', 'Bilgisayar Bilimi', 1, 1, 4, 6, 3, 2, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BB201', 'Veri Yapıları', 'Fen Fakültesi', 'Bilgisayar Bilimi', 2, 3, 4, 6, 3, 2, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BB205', 'Ayrık Matematik', 'Fen Fakültesi', 'Bilgisayar Bilimi', 2, 3, 3, 5, 3, 0, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BB301', 'İşletim Sistemleri', 'Fen Fakültesi', 'Bilgisayar Bilimi', 3, 5, 4, 6, 3, 2, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BB305', 'Veritabanı Yönetim Sistemleri', 'Fen Fakültesi', 'Bilgisayar Bilimi', 3, 5, 4, 6, 3, 2, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BLM101', 'Bilgisayar Mühendisliğine Giriş', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 1, 1, 3, 5, 3, 0, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active'),
(gen_random_uuid(), 'BLM103', 'Programlama I', 'Mühendislik Fakültesi', 'Bilgisayar Mühendisliği', 1, 1, 4, 6, 3, 2, 'mandatory', 'theoretical', 'undergraduate', 'on_campus', 'Türkçe', 'active')
ON CONFLICT (course_code) DO NOTHING;

-- Optionally, insert a dummy staff so we can open semester courses
INSERT INTO staff.staff (
    id, email, first_name, last_name, role, department, is_active
) VALUES (
    '11111111-1111-1111-1111-111111111111', 'prof.veli@university.edu.tr', 'Veli', 'Hoca', 'instructor', 'Bilgisayar Bilimi', true
);

-- Open some courses for the current semester (e.g., 2025_spring)
INSERT INTO course_catalog.semester_courses (
    semester, course_code, credits, class_level, instructor_id, instructor_fullname, classroom_location, max_capacity
) VALUES
('2025_spring', 'BB101', 3, 1, '11111111-1111-1111-1111-111111111111', 'Veli Hoca', 'Fen-A-101', 100),
('2025_spring', 'BB103', 4, 1, '11111111-1111-1111-1111-111111111111', 'Veli Hoca', 'Fen-Lab-1', 50),
('2025_spring', 'BB201', 4, 2, '11111111-1111-1111-1111-111111111111', 'Veli Hoca', 'Fen-A-201', 80),
('2025_spring', 'BLM101', 3, 1, '11111111-1111-1111-1111-111111111111', 'Veli Hoca', 'Müh-101', 120)
ON CONFLICT (semester, course_code) DO NOTHING;
