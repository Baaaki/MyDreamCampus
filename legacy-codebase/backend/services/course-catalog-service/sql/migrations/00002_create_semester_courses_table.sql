-- +goose Up
-- Enum definition (consistent with Enrollment Service)
CREATE TYPE day_of_week_enum AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

-- Single table for all semesters (no partitioning)
CREATE TABLE IF NOT EXISTS semester_courses (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    semester VARCHAR(50) NOT NULL,            -- "2025_spring", "2025_fall", "2026_spring"
    course_code VARCHAR(50) NOT NULL REFERENCES course_catalog(course_code),
    credits SMALLINT NOT NULL CHECK (credits > 0 AND credits <= 30),  -- Cached from course_catalog (immutable snapshot for ECTS calculation)
    class_level SMALLINT NOT NULL CHECK (class_level BETWEEN 1 AND 6),  -- Hangi sınıf seviyesi için açık (1-6)
    instructor_id UUID NOT NULL,              -- Staff Service UUID (system reference)
    instructor_fullname VARCHAR(150) NOT NULL, -- Cached: first_name + last_name from Staff Service (immutable snapshot)
    classroom_location VARCHAR(100) NOT NULL, -- "A Blok 301"
    max_capacity SMALLINT NOT NULL CHECK (max_capacity > 0 AND max_capacity <= 1000),  -- Max 1000 kişilik amfi

    -- Snapshot of course prerequisites at the time of semester course creation
    -- This ensures enrollment rules don't change retroactively when catalog is updated
    -- Format: [{"course_code":"MAT101","min_grade":60}, ...]
    prerequisites JSONB NOT NULL DEFAULT '[]',

    -- Sınav Yapısı Konfigürasyonu (Assessment Schema)
    -- Örnek: [{"slug": "midterm", "name": "Vize", "weight": 40}, {"slug": "final", "name": "Final", "weight": 60}]
    assessment_schema JSONB NOT NULL DEFAULT '[{"slug": "midterm", "name": "Vize", "weight": 40},
                                               {"slug": "final", "name": "Final", "weight": 60}]',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Prevent duplicate: Same course, same semester
    UNIQUE(semester, course_code),

    -- Validation constraints
    CONSTRAINT chk_prerequisites_valid CHECK (
        jsonb_typeof(prerequisites) = 'array'
    ),
    CONSTRAINT chk_assessment_schema_valid CHECK (
        jsonb_typeof(assessment_schema) = 'array'
        AND jsonb_array_length(assessment_schema) > 0
    )
);

CREATE INDEX idx_semester_courses_semester ON semester_courses(semester);
CREATE INDEX idx_semester_courses_course_code ON semester_courses(course_code);
CREATE INDEX idx_semester_courses_instructor ON semester_courses(instructor_id);
CREATE INDEX idx_semester_courses_semester_code ON semester_courses(semester, course_code);

-- +goose Down
DROP TABLE IF EXISTS semester_courses;
DROP TYPE IF EXISTS day_of_week_enum;
