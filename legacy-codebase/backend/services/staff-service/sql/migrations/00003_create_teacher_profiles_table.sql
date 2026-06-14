-- +goose Up
CREATE TABLE IF NOT EXISTS teacher_profiles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    staff_id UUID NOT NULL UNIQUE REFERENCES staff(id) ON DELETE CASCADE,

    -- Academic title (Prof. Dr., Doç. Dr., Dr. Öğr. Üyesi, vb.)
    academic_title VARCHAR(50),

    -- Faculty info (might differ from staff department for display purposes)
    faculty VARCHAR(200),

    -- Profile image URL
    profile_image_url TEXT,

    -- Education history as JSONB array
    -- [{id, degree, institution, department, year}]
    education JSONB DEFAULT '[]'::jsonb,

    -- Articles/Publications as JSONB array
    -- [{id, title, journal, year, authors, doi, journalType, domesticInternational, language, articleType}]
    articles JSONB DEFAULT '[]'::jsonb,

    -- Conference bulletins as JSONB array
    -- [{id, title, conference, year, location}]
    bulletins JSONB DEFAULT '[]'::jsonb,

    -- Research projects as JSONB array
    -- [{id, title, role, funder, startYear, endYear, status}]
    projects JSONB DEFAULT '[]'::jsonb,

    -- Awards as JSONB array
    -- [{id, title, institution, year}]
    awards JSONB DEFAULT '[]'::jsonb,

    -- Scholarships as JSONB array
    -- [{id, title, institution, year}]
    scholarships JSONB DEFAULT '[]'::jsonb,

    -- Administrative assignments as JSONB array
    -- [{id, title, institution, startYear, endYear}]
    admin_assignments JSONB DEFAULT '[]'::jsonb,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_teacher_profiles_staff_id ON teacher_profiles(staff_id);

-- +goose Down
DROP TABLE IF EXISTS teacher_profiles;
