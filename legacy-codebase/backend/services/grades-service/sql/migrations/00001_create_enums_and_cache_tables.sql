-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ENUM Tanımları
CREATE TYPE grading_type_enum AS ENUM ('absolute', 'relative');
CREATE TYPE grade_point_enum AS ENUM (
    '4.00', '3.75', '3.50', '3.25', '3.00',
    '2.75', '2.50', '2.25', '2.00', '1.75',
    '1.50', '1.25', '1.00', '0.50', '0.00'
);

-- ==========================================
-- A. CACHE TABLOLARI
-- ==========================================

CREATE TABLE students_cache (
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

CREATE INDEX idx_students_cache_number ON students_cache(student_number);
CREATE INDEX idx_students_cache_active ON students_cache(is_active) WHERE is_active = true;

CREATE TABLE courses_cache (
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

CREATE INDEX idx_courses_cache_semester ON courses_cache(semester);
CREATE INDEX idx_courses_cache_instructor ON courses_cache(instructor_id);

CREATE TABLE prerequisite_courses_cache (
    course_code VARCHAR(50) NOT NULL,
    course_id UUID NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (course_code, course_id)
);

-- +goose Down
DROP TABLE IF EXISTS prerequisite_courses_cache;
DROP TABLE IF EXISTS courses_cache;
DROP TABLE IF EXISTS students_cache;
DROP TYPE IF EXISTS grade_point_enum;
DROP TYPE IF EXISTS grading_type_enum;
DROP EXTENSION IF EXISTS "pgcrypto";
