-- +goose Up
CREATE TABLE IF NOT EXISTS cafeterias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    has_vegan_menu BOOLEAN NOT NULL DEFAULT false,
    serves_dinner BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cafeterias_is_active ON cafeterias(is_active) WHERE is_active = true;

-- +goose Down
DROP TABLE IF EXISTS cafeterias;
