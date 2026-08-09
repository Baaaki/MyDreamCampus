-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ENUM Tanımları
CREATE TYPE grades.grading_type_enum AS ENUM ('absolute', 'relative');
CREATE TYPE grades.grade_point_enum AS ENUM (
    '4.00', '3.75', '3.50', '3.25', '3.00',
    '2.75', '2.50', '2.25', '2.00', '1.75',
    '1.50', '1.25', '1.00', '0.50', '0.00'
);

-- ==========================================
-- A. VIEWS (Replaced Caches)
-- ==========================================

CREATE TABLE grades.students_view (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    department VARCHAR(100),
    class_level SMALLINT,
    is_active BOOLEAN DEFAULT TRUE,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_grades_students_view_number ON grades.students_view(student_number);
CREATE INDEX idx_grades_students_view_active ON grades.students_view(is_active) WHERE is_active = true;

CREATE TABLE grades.courses_view (
    id UUID PRIMARY KEY,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,
    department VARCHAR(100),
    instructor_id UUID NOT NULL,
    instructor_fullname VARCHAR(150),
    assessment_schema JSONB NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_grades_courses_view_semester ON grades.courses_view(semester);
CREATE INDEX idx_grades_courses_view_instructor ON grades.courses_view(instructor_id);

CREATE TABLE grades.prerequisite_courses_view (
    course_code VARCHAR(50) NOT NULL,
    course_id UUID NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (course_code, course_id)
);

-- ==========================================
-- B. OPERASYONEL TABLOLAR (Dönem İçi)
-- ==========================================

CREATE TABLE grades.student_course_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES grades.students_view(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES grades.courses_view(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    is_attendance_failed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_grades_registrations_student ON grades.student_course_registrations(student_id);
CREATE INDEX idx_grades_registrations_course ON grades.student_course_registrations(course_id);

CREATE TABLE grades.student_assessment_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_id UUID NOT NULL REFERENCES grades.student_course_registrations(id) ON DELETE CASCADE,
    slug VARCHAR(50) NOT NULL,
    score DECIMAL(5,2) CHECK (score >= 0 AND score <= 100),
    is_absent BOOLEAN DEFAULT FALSE,
    graded_by UUID NOT NULL,
    graded_at TIMESTAMP DEFAULT NOW(),
    is_locked BOOLEAN DEFAULT FALSE,
    UNIQUE(registration_id, slug)
);

CREATE INDEX idx_grades_scores_registration ON grades.student_assessment_scores(registration_id);

-- ==========================================
-- C. TAMAMLANMIŞ DERSLER (Kalıcı)
-- ==========================================

CREATE TABLE grades.student_completed_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Öğrenci Snapshot
    student_id UUID NOT NULL,
    student_number VARCHAR(50) NOT NULL,
    student_first_name VARCHAR(100) NOT NULL,
    student_last_name VARCHAR(100) NOT NULL,
    student_department VARCHAR(100),

    -- Ders Snapshot
    course_id UUID NOT NULL,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,

    -- Hoca Snapshot
    instructor_id UUID NOT NULL,
    instructor_name VARCHAR(150) NOT NULL,

    -- Notlar
    assessment_scores JSONB NOT NULL,

    -- Hesaplanan Sonuç
    weighted_average DECIMAL(5,2) NOT NULL,
    grade_point grades.grade_point_enum NOT NULL,

    -- Notlandırma Bilgisi
    grading_type grades.grading_type_enum NOT NULL,
    grading_config JSONB,
    class_statistics JSONB,

    -- Devamsızlık Bilgisi
    is_attendance_failed BOOLEAN DEFAULT FALSE,

    finalized_at TIMESTAMP NOT NULL,
    finalized_by UUID NOT NULL,

    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_grades_completed_student ON grades.student_completed_courses(student_id);
CREATE INDEX idx_grades_completed_semester ON grades.student_completed_courses(semester);
CREATE INDEX idx_grades_completed_course_code ON grades.student_completed_courses(course_code);
CREATE INDEX idx_grades_completed_student_course_code ON grades.student_completed_courses(student_id, course_code);

-- +goose Down
DROP TABLE IF EXISTS grades.student_completed_courses;
DROP TABLE IF EXISTS grades.student_assessment_scores;
DROP TABLE IF EXISTS grades.student_course_registrations;
DROP TABLE IF EXISTS grades.prerequisite_courses_view;
DROP TABLE IF EXISTS grades.courses_view;
DROP TABLE IF EXISTS grades.students_view;
DROP TYPE IF EXISTS grades.grade_point_enum;
DROP TYPE IF EXISTS grades.grading_type_enum;
