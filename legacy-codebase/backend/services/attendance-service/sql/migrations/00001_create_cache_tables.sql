-- +goose Up
-- ==========================================
-- A. CACHE TABLOLARI (Diğer Servislerden)
-- ==========================================

CREATE TABLE students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    department VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_cache_number ON students_cache(student_number);
CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;

-- Ders Bilgileri (Course Catalog'dan)
CREATE TABLE courses_cache (
    id UUID PRIMARY KEY,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,
    department VARCHAR(100),
    instructor_id UUID NOT NULL,
    instructor_fullname VARCHAR(150),
    total_weeks SMALLINT DEFAULT 14,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_courses_cache_semester ON courses_cache(semester);
CREATE INDEX idx_courses_cache_instructor ON courses_cache(instructor_id);

-- Öğrenci-Ders Kayıtları (Enrollment'tan)
CREATE TABLE enrollments_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_enrollments_student ON enrollments_cache(student_id);
CREATE INDEX idx_enrollments_course ON enrollments_cache(course_id);
CREATE INDEX idx_enrollments_semester ON enrollments_cache(semester);

-- +goose Down
DROP TABLE IF EXISTS enrollments_cache;
DROP TABLE IF EXISTS courses_cache;
DROP TABLE IF EXISTS students_cache;
