-- +goose Up
-- Enum definition for course catalog status
CREATE TYPE course_catalog.course_catalog_status_enum AS ENUM (
    'active',           -- Aktif ders (öğrencilere açılabilir)
    'draft',            -- Taslak (henüz onaylanmamış)
    'pending_approval', -- Onay bekliyor
    'under_revision',   -- Revizyon aşamasında
    'archived',         -- Arşivlenmiş (artık açılmaz, eski kayıtlar için)
    'suspended'         -- Askıya alınmış (geçici olarak kapatılmış)
);

-- Enum definition for course type (zorunlu/seçmeli)
CREATE TYPE course_catalog.course_type_enum AS ENUM (
    'mandatory',        -- Zorunlu ders
    'elective'          -- Seçmeli ders
);

-- Enum definition for course category (ders kategorisi)
CREATE TYPE course_catalog.course_category_enum AS ENUM (
    'theoretical',      -- Normal teorik ders
    'practical',        -- Uygulama ağırlıklı ders
    'internship',       -- Staj
    'project',          -- Bitirme projesi / Proje dersi
    'seminar'           -- Seminer
);

-- Enum definition for education level
CREATE TYPE course_catalog.education_level_enum AS ENUM (
    'undergraduate',    -- Lisans
    'graduate',         -- Yüksek Lisans
    'doctorate'         -- Doktora
);

-- Enum definition for teaching type
CREATE TYPE course_catalog.teaching_type_enum AS ENUM (
    'on_campus',        -- Örgün Öğretim
    'online',           -- Uzaktan Öğretim
    'hybrid'            -- Hibrit (Karma)
);

CREATE TABLE IF NOT EXISTS course_catalog.course_catalog (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    course_code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,

    -- Organizasyonel bilgiler
    faculty VARCHAR(100) NOT NULL,                  -- Fakülte
    department VARCHAR(100) NOT NULL,               -- Bölüm
    offering_unit VARCHAR(255),                     -- Dersi veren birim (ortak dersler için, örn: Atatürk İlkeleri Enstitüsü)

    -- Dönem ve seviye bilgileri
    class_level SMALLINT NOT NULL CHECK (class_level BETWEEN 1 AND 6),  -- Sınıf seviyesi (1-6)
    semester SMALLINT CHECK (semester BETWEEN 1 AND 8),                  -- Dönem (1-8, Güz/Bahar)

    -- Kredi ve saat bilgileri
    credits SMALLINT NOT NULL CHECK (credits >= 0 AND credits <= 30),    -- Yerel kredi (0 olabilir, staj için)
    ects SMALLINT CHECK (ects >= 0 AND ects <= 60),                      -- ECTS kredisi
    theoretical_hours SMALLINT NOT NULL DEFAULT 0 CHECK (theoretical_hours >= 0 AND theoretical_hours <= 20),
    practical_hours SMALLINT NOT NULL DEFAULT 0 CHECK (practical_hours >= 0 AND practical_hours <= 20),
    lab_hours SMALLINT NOT NULL DEFAULT 0 CHECK (lab_hours >= 0 AND lab_hours <= 20),

    -- Ders tipleri ve kategorileri
    course_type course_catalog.course_type_enum NOT NULL DEFAULT 'mandatory',
    course_category course_catalog.course_category_enum NOT NULL DEFAULT 'theoretical',
    education_level course_catalog.education_level_enum NOT NULL DEFAULT 'undergraduate',
    teaching_type course_catalog.teaching_type_enum NOT NULL DEFAULT 'on_campus',
    language VARCHAR(50) NOT NULL DEFAULT 'Türkçe',

    -- Önkoşullar (denormalized for read performance)
    -- [{id, course_code, course_name}, ...]
    prerequisites JSONB DEFAULT '[]',

    -- Ders koordinatörü bilgileri
    -- {title, name, email, phone, office}
    coordinator JSONB,

    -- Ders içeriği
    purpose TEXT,                                   -- Dersin amacı
    description TEXT,                               -- Ders tanımı
    learning_outcomes TEXT,                         -- Öğrenim çıktıları (metin olarak)

    -- Detaylı içerik (JSONB arrays)
    -- ["Kazanım 1", "Kazanım 2", ...]
    learning_outcomes_list JSONB DEFAULT '[]',

    -- [{week: 1, topic: "Konu başlığı"}, ...]
    weekly_topics JSONB DEFAULT '[]',

    -- ["Kaynak 1", "Kaynak 2", ...]
    recommended_sources JSONB DEFAULT '[]',

    syllabus TEXT,                                  -- Ders içeriği / müfredat (metin olarak)

    -- Durum ve zaman damgaları
    status course_catalog.course_catalog_status_enum NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Primary indexes
CREATE INDEX idx_catalog_code ON course_catalog.course_catalog(course_code);
CREATE INDEX idx_catalog_department ON course_catalog.course_catalog(department);
CREATE INDEX idx_catalog_faculty ON course_catalog.course_catalog(faculty);

-- Filtering indexes
CREATE INDEX idx_catalog_status ON course_catalog.course_catalog(status);
CREATE INDEX idx_catalog_course_type ON course_catalog.course_catalog(course_type);
CREATE INDEX idx_catalog_course_category ON course_catalog.course_catalog(course_category);
CREATE INDEX idx_catalog_class_level ON course_catalog.course_catalog(class_level);
CREATE INDEX idx_catalog_semester ON course_catalog.course_catalog(semester);
CREATE INDEX idx_catalog_education_level ON course_catalog.course_catalog(education_level);
CREATE INDEX idx_catalog_language ON course_catalog.course_catalog(language);

-- Composite indexes for common queries
CREATE INDEX idx_catalog_dept_semester ON course_catalog.course_catalog(department, semester);
CREATE INDEX idx_catalog_dept_class_level ON course_catalog.course_catalog(department, class_level);
CREATE INDEX idx_catalog_faculty_dept ON course_catalog.course_catalog(faculty, department);

-- GIN indexes for JSONB columns
CREATE INDEX idx_catalog_prerequisites_gin ON course_catalog.course_catalog USING GIN(prerequisites);
CREATE INDEX idx_catalog_weekly_topics_gin ON course_catalog.course_catalog USING GIN(weekly_topics);

-- +goose Down
DROP TABLE IF EXISTS course_catalog.course_catalog;
DROP TYPE IF EXISTS course_catalog.teaching_type_enum;
DROP TYPE IF EXISTS course_catalog.education_level_enum;
DROP TYPE IF EXISTS course_catalog.course_category_enum;
DROP TYPE IF EXISTS course_catalog.course_type_enum;
DROP TYPE IF EXISTS course_catalog.course_catalog_status_enum;
