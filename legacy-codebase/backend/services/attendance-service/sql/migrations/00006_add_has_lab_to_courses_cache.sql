-- +goose Up
ALTER TABLE courses_cache ADD COLUMN has_lab BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE courses_cache DROP COLUMN IF EXISTS has_lab;
