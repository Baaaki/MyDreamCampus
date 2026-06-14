-- +goose Up
-- Create custom enum types
CREATE TYPE enrollment_status_enum AS ENUM ('pending', 'approved');
CREATE TYPE day_of_week_enum AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');
CREATE TYPE course_type_enum AS ENUM ('mandatory', 'elective');

-- Local student cache (synced from Student Service via RabbitMQ events)
CREATE TABLE IF NOT EXISTS students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    department VARCHAR(100),
    class_level SMALLINT CHECK (class_level BETWEEN 1 AND 6),
    advisor_id UUID,
    status VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;
CREATE INDEX idx_students_cache_advisor ON students_cache(advisor_id);

-- +goose Down
DROP TABLE IF EXISTS students_cache;
DROP TYPE IF EXISTS course_type_enum;
DROP TYPE IF EXISTS day_of_week_enum;
DROP TYPE IF EXISTS enrollment_status_enum;
