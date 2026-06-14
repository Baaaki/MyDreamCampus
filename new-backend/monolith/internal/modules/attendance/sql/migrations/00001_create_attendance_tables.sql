-- +goose Up
-- ==========================================
-- A. VIEW TABLOLARI (Diğer Servislerden)
-- ==========================================

CREATE TABLE attendance.students_view (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    department VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_students_view_number ON attendance.students_view(student_number);
CREATE INDEX idx_students_view_is_active ON attendance.students_view(is_active) WHERE is_active = true;

CREATE TABLE attendance.courses_view (
    id UUID PRIMARY KEY,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,
    department VARCHAR(100),
    instructor_id UUID NOT NULL,
    instructor_fullname VARCHAR(150),
    total_weeks SMALLINT DEFAULT 14,
    has_lab BOOLEAN NOT NULL DEFAULT false,
    synced_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_courses_view_semester ON attendance.courses_view(semester);
CREATE INDEX idx_courses_view_instructor ON attendance.courses_view(instructor_id);

CREATE TABLE attendance.enrollments_view (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES attendance.students_view(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES attendance.courses_view(id) ON DELETE CASCADE,
    semester VARCHAR(50) NOT NULL,
    synced_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_enrollments_student ON attendance.enrollments_view(student_id);
CREATE INDEX idx_enrollments_course ON attendance.enrollments_view(course_id);
CREATE INDEX idx_enrollments_semester ON attendance.enrollments_view(semester);

-- ==========================================
-- B. OPERASYONEL TABLOLAR
-- ==========================================

CREATE TYPE attendance.session_type_enum AS ENUM ('theory', 'lab');

CREATE TABLE attendance.attendance_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES attendance.courses_view(id) ON DELETE CASCADE,
    instructor_id UUID NOT NULL,
    semester VARCHAR(50) NOT NULL,
    week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 14),
    session_date DATE NOT NULL,
    session_type attendance.session_type_enum NOT NULL DEFAULT 'theory',
    qr_secret VARCHAR(64) NOT NULL,
    qr_rotation_interval SMALLINT DEFAULT 15,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(course_id, week_number, session_type)
);

CREATE INDEX idx_sessions_course ON attendance.attendance_sessions(course_id);
CREATE INDEX idx_sessions_active ON attendance.attendance_sessions(is_active, expires_at) WHERE is_active = TRUE;
CREATE INDEX idx_sessions_semester ON attendance.attendance_sessions(semester);
CREATE INDEX idx_sessions_session_type ON attendance.attendance_sessions(session_type);

CREATE TABLE attendance.attendance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES attendance.attendance_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES attendance.students_view(id) ON DELETE CASCADE,
    course_id UUID NOT NULL,
    semester VARCHAR(50) NOT NULL,
    week_number SMALLINT NOT NULL,
    session_type attendance.session_type_enum NOT NULL DEFAULT 'theory',
    marked_via VARCHAR(20) NOT NULL CHECK (marked_via IN ('qr_scan', 'manual', 'admin')),
    scanned_at TIMESTAMP,
    qr_timestamp BIGINT,
    manually_marked_by UUID,
    manually_marked_at TIMESTAMP,
    manual_note TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(session_id, student_id)
);

CREATE INDEX idx_records_session ON attendance.attendance_records(session_id);
CREATE INDEX idx_records_student ON attendance.attendance_records(student_id);
CREATE INDEX idx_records_student_course ON attendance.attendance_records(student_id, course_id);
CREATE INDEX idx_records_course_semester ON attendance.attendance_records(course_id, semester);
CREATE INDEX idx_records_week ON attendance.attendance_records(course_id, week_number);
CREATE INDEX idx_records_session_type ON attendance.attendance_records(session_type);

CREATE TABLE attendance.academic_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semester VARCHAR(50) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_academic_periods_semester ON attendance.academic_periods(semester);

-- +goose Down
DROP TABLE IF EXISTS attendance.academic_periods;
DROP TABLE IF EXISTS attendance.attendance_records;
DROP TABLE IF EXISTS attendance.attendance_sessions;
DROP TYPE IF EXISTS attendance.session_type_enum;
DROP TABLE IF EXISTS attendance.enrollments_view;
DROP TABLE IF EXISTS attendance.courses_view;
DROP TABLE IF EXISTS attendance.students_view;
