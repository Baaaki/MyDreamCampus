-- +goose Up
-- ==========================================
-- B. OPERASYONEL TABLOLAR
-- ==========================================

-- Yoklama Oturumları (Hoca başlatır)
CREATE TABLE attendance_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses_cache(id) ON DELETE CASCADE,
    instructor_id UUID NOT NULL,
    semester VARCHAR(50) NOT NULL,
    week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 14),
    session_date DATE NOT NULL,

    -- QR Security
    qr_secret VARCHAR(64) NOT NULL,
    qr_rotation_interval SMALLINT DEFAULT 15,

    -- Session durumu
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT NOW(),

    -- Aynı ders için aynı hafta sadece 1 session
    UNIQUE(course_id, week_number)
);

CREATE INDEX idx_sessions_course ON attendance_sessions(course_id);
CREATE INDEX idx_sessions_active ON attendance_sessions(is_active, expires_at) WHERE is_active = TRUE;
CREATE INDEX idx_sessions_semester ON attendance_sessions(semester);

-- Yoklama Kayıtları (kayıt var = derse geldi, kayıt yok = gelmedi)
CREATE TABLE attendance_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES attendance_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    course_id UUID NOT NULL,
    semester VARCHAR(50) NOT NULL,
    week_number SMALLINT NOT NULL,

    -- Nasıl işaretlendi
    marked_via VARCHAR(20) NOT NULL CHECK (marked_via IN ('qr_scan', 'manual', 'admin')),

    -- QR scan detayları (sadece qr_scan için)
    scanned_at TIMESTAMP,
    qr_timestamp BIGINT,

    -- Manuel/admin giriş detayları
    manually_marked_by UUID,
    manually_marked_at TIMESTAMP,
    manual_note TEXT,

    created_at TIMESTAMP DEFAULT NOW(),

    -- Bir öğrenci bir session'da sadece 1 kayıt
    UNIQUE(session_id, student_id)
);

CREATE INDEX idx_records_session ON attendance_records(session_id);
CREATE INDEX idx_records_student ON attendance_records(student_id);
CREATE INDEX idx_records_student_course ON attendance_records(student_id, course_id);
CREATE INDEX idx_records_course_semester ON attendance_records(course_id, semester);
CREATE INDEX idx_records_week ON attendance_records(course_id, week_number);

-- +goose Down
DROP TABLE IF EXISTS attendance_records;
DROP TABLE IF EXISTS attendance_sessions;
