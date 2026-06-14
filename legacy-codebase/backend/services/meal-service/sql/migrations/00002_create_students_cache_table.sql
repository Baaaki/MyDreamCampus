-- +goose Up
CREATE TABLE IF NOT EXISTS students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_students_cache_student_number ON students_cache(student_number);
CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;

-- +goose Down
DROP TABLE IF EXISTS students_cache;
