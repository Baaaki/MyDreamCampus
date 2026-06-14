-- +goose Up
-- Local semester course cache (synced from Course Catalog via RabbitMQ events)
CREATE TABLE IF NOT EXISTS semester_courses_cache (
    id UUID PRIMARY KEY,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255),
    faculty VARCHAR(100),
    department VARCHAR(100),
    credits SMALLINT NOT NULL,
    course_type course_type_enum NOT NULL,
    class_level SMALLINT CHECK (class_level BETWEEN 1 AND 6),
    semester VARCHAR(50) NOT NULL,
    instructor_id UUID,
    instructor_fullname VARCHAR(150),
    classroom_location VARCHAR(100),
    max_capacity SMALLINT NOT NULL,
    current_enrollment SMALLINT DEFAULT 0 CHECK (current_enrollment >= 0),
    prerequisites JSONB DEFAULT '[]',
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_semester_courses_cache_department_semester ON semester_courses_cache(department, semester);
CREATE INDEX idx_semester_courses_cache_class_level ON semester_courses_cache(class_level);

-- +goose Down
DROP TABLE IF EXISTS semester_courses_cache;
