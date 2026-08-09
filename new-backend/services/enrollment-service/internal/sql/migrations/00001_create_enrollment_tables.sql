-- +goose Up
-- Create custom enum types
CREATE TYPE enrollment.enrollment_status_enum AS ENUM ('pending', 'approved');

-- Enrollment programs (sadece pending ve approved programlar)
CREATE TABLE IF NOT EXISTS enrollment.enrollment_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL, -- Logical FK to student.students
    semester VARCHAR(50) NOT NULL,
    status enrollment.enrollment_status_enum DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(student_id, semester)
);

CREATE INDEX idx_programs_student ON enrollment.enrollment_programs(student_id);
CREATE INDEX idx_programs_status ON enrollment.enrollment_programs(status);
CREATE INDEX idx_programs_semester ON enrollment.enrollment_programs(semester);

-- Individual courses in a program
CREATE TABLE IF NOT EXISTS enrollment.enrollment_program_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES enrollment.enrollment_programs(id) ON DELETE CASCADE,
    course_id UUID NOT NULL, -- Logical FK to course_catalog.courses
    -- Snapshot fields for Strateji 2 (historical accuracy)
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(program_id, course_id)
);

-- Rejection logs (reddedilen programların tarihçesi)
CREATE TABLE IF NOT EXISTS enrollment.enrollment_rejection_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_program_id UUID NOT NULL,
    student_id UUID NOT NULL, -- Logical FK to student.students
    advisor_id UUID NOT NULL, -- Logical FK to staff.staff
    advisor_fullname VARCHAR(150) NOT NULL, -- Snapshot
    semester VARCHAR(50) NOT NULL,
    rejection_reason TEXT NOT NULL,
    rejected_courses JSONB NOT NULL,
    rejected_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rejection_logs_student ON enrollment.enrollment_rejection_logs(student_id);
CREATE INDEX idx_rejection_logs_student_semester ON enrollment.enrollment_rejection_logs(student_id, semester);

-- +goose Down
DROP TABLE IF EXISTS enrollment.enrollment_rejection_logs;
DROP TABLE IF EXISTS enrollment.enrollment_program_courses;
DROP TABLE IF EXISTS enrollment.enrollment_programs;
DROP TYPE IF EXISTS enrollment.enrollment_status_enum;
