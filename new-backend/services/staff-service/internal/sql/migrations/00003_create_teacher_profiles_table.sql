-- +goose Up
CREATE TABLE IF NOT EXISTS staff.teacher_profiles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    staff_id UUID NOT NULL UNIQUE REFERENCES staff.staff(id) ON DELETE CASCADE,

    -- Academic title (Prof. Dr., Doc. Dr., Dr. Ogr. Uyesi, etc.)
    academic_title VARCHAR(50),

    -- Faculty info (might differ from staff department for display purposes)
    faculty VARCHAR(200),

    -- Profile image URL
    profile_image_url TEXT,

    -- Education history as JSONB array
    education JSONB DEFAULT '[]'::jsonb,

    -- Articles/Publications as JSONB array
    articles JSONB DEFAULT '[]'::jsonb,

    -- Conference bulletins as JSONB array
    bulletins JSONB DEFAULT '[]'::jsonb,

    -- Research projects as JSONB array
    projects JSONB DEFAULT '[]'::jsonb,

    -- Awards as JSONB array
    awards JSONB DEFAULT '[]'::jsonb,

    -- Scholarships as JSONB array
    scholarships JSONB DEFAULT '[]'::jsonb,

    -- Administrative assignments as JSONB array
    admin_assignments JSONB DEFAULT '[]'::jsonb,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_teacher_profiles_staff_id ON staff.teacher_profiles(staff_id);

-- +goose Down
DROP TABLE IF EXISTS staff.teacher_profiles;
