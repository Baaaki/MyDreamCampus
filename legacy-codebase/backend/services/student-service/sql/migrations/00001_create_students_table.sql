-- +goose Up
CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    faculty VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    enrollment_year INT NOT NULL,
    class_level SMALLINT NOT NULL DEFAULT 1 CHECK (class_level BETWEEN 1 AND 6),
    advisor_id UUID,
    status VARCHAR(50) DEFAULT 'active',
    is_active BOOLEAN DEFAULT true,
    deleted_at TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Unique constraints only for active students
CREATE UNIQUE INDEX idx_students_number_unique
    ON students(student_number) WHERE is_active = true;

CREATE UNIQUE INDEX idx_students_email_unique
    ON students(email) WHERE is_active = true;

-- Performance indexes (only active students)
CREATE INDEX idx_students_department ON students(department) WHERE is_active = true;
CREATE INDEX idx_students_status ON students(status) WHERE is_active = true;
CREATE INDEX idx_students_class_level ON students(class_level) WHERE is_active = true;
CREATE INDEX idx_students_advisor ON students(advisor_id) WHERE is_active = true;
CREATE INDEX idx_students_is_active ON students(is_active);

-- Full-text search index
CREATE INDEX idx_students_fulltext ON students
    USING gin(to_tsvector('english', first_name || ' ' || last_name || ' ' || student_number));

-- Compound index for common queries
CREATE INDEX idx_students_dept_class ON students(department, class_level) WHERE is_active = true;

-- +goose Down
DROP TABLE IF EXISTS students;
