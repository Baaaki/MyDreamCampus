-- +goose Up
-- Student passed prerequisites (prerequisite validation için)
-- Sadece GEÇİLEN önkoşul dersler kaydedilir
CREATE TABLE IF NOT EXISTS student_passed_prerequisites (
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_code VARCHAR(50) NOT NULL,
    semester VARCHAR(50) NOT NULL,
    grade_point VARCHAR(10),
    synced_at TIMESTAMP DEFAULT NOW(),

    PRIMARY KEY (student_id, course_code)
);

-- +goose Down
DROP TABLE IF EXISTS student_passed_prerequisites;
