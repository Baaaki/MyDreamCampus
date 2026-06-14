-- +goose Up
-- Rejection logs (reddedilen programların tarihçesi)
CREATE TABLE IF NOT EXISTS enrollment_rejection_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_program_id UUID NOT NULL,
    student_id UUID NOT NULL REFERENCES students_cache(id),
    advisor_id UUID NOT NULL,
    advisor_fullname VARCHAR(150) NOT NULL,
    semester VARCHAR(50) NOT NULL,
    rejection_reason TEXT NOT NULL,
    rejected_courses JSONB NOT NULL,
    rejected_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rejection_logs_student ON enrollment_rejection_logs(student_id);
CREATE INDEX idx_rejection_logs_student_semester ON enrollment_rejection_logs(student_id, semester);

-- +goose Down
DROP TABLE IF EXISTS enrollment_rejection_logs;
