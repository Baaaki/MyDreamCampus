-- +goose Up
-- Enrollment programs (sadece pending ve approved programlar)
CREATE TABLE IF NOT EXISTS enrollment_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id),
    semester VARCHAR(50) NOT NULL,
    status enrollment_status_enum DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(student_id, semester)
);

CREATE INDEX idx_programs_student ON enrollment_programs(student_id);
CREATE INDEX idx_programs_status ON enrollment_programs(status);
CREATE INDEX idx_programs_semester ON enrollment_programs(semester);

-- +goose Down
DROP TABLE IF EXISTS enrollment_programs;
