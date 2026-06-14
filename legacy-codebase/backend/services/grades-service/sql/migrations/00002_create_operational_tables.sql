-- +goose Up
-- ==========================================
-- B. OPERASYONEL TABLOLAR (Dönem İçi)
-- ==========================================

CREATE TABLE student_course_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    is_attendance_failed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_registrations_student ON student_course_registrations(student_id);
CREATE INDEX idx_registrations_course ON student_course_registrations(course_id);

CREATE TABLE student_assessment_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    registration_id UUID NOT NULL REFERENCES student_course_registrations(id) ON DELETE CASCADE,
    slug VARCHAR(50) NOT NULL,
    score DECIMAL(5,2) CHECK (score >= 0 AND score <= 100),
    is_absent BOOLEAN DEFAULT FALSE,
    graded_by UUID NOT NULL,
    graded_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(registration_id, slug)
);

CREATE INDEX idx_scores_registration ON student_assessment_scores(registration_id);

-- +goose Down
DROP TABLE IF EXISTS student_assessment_scores;
DROP TABLE IF EXISTS student_course_registrations;
