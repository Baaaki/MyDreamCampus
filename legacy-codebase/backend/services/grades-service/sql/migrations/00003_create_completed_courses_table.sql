-- +goose Up
-- ==========================================
-- C. TAMAMLANMIŞ DERSLER (Kalıcı)
-- ==========================================

CREATE TABLE student_completed_courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Öğrenci Snapshot
    student_id UUID NOT NULL,
    student_number VARCHAR(50) NOT NULL,
    student_first_name VARCHAR(100) NOT NULL,
    student_last_name VARCHAR(100) NOT NULL,
    student_department VARCHAR(100),

    -- Ders Snapshot
    course_id UUID NOT NULL,
    course_code VARCHAR(50) NOT NULL,
    course_name VARCHAR(255) NOT NULL,
    credits SMALLINT NOT NULL,
    semester VARCHAR(50) NOT NULL,

    -- Hoca Snapshot
    instructor_id UUID NOT NULL,
    instructor_name VARCHAR(150) NOT NULL,

    -- Notlar
    assessment_scores JSONB NOT NULL,

    -- Hesaplanan Sonuç
    weighted_average DECIMAL(5,2) NOT NULL,
    grade_point grade_point_enum NOT NULL,

    -- Notlandırma Bilgisi
    grading_type grading_type_enum NOT NULL,
    grading_config JSONB,
    class_statistics JSONB,

    -- Devamsızlık Bilgisi
    is_attendance_failed BOOLEAN DEFAULT FALSE,

    finalized_at TIMESTAMP NOT NULL,
    finalized_by UUID NOT NULL,

    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_completed_student ON student_completed_courses(student_id);
CREATE INDEX idx_completed_semester ON student_completed_courses(semester);
CREATE INDEX idx_completed_course_code ON student_completed_courses(course_code);
CREATE INDEX idx_completed_student_course_code ON student_completed_courses(student_id, course_code);

-- +goose Down
DROP TABLE IF EXISTS student_completed_courses;
