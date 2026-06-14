-- +goose Up
CREATE TABLE IF NOT EXISTS monthly_menus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    year SMALLINT NOT NULL,
    month SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12),
    menu_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_year_month UNIQUE(year, month)
);

CREATE INDEX idx_monthly_menus_year_month ON monthly_menus(year, month);

-- +goose Down
DROP TABLE IF EXISTS monthly_menus;
