-- +goose Up
ALTER TABLE auth.users
    ADD COLUMN is_superadmin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_demo BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE auth.users
    DROP COLUMN is_demo,
    DROP COLUMN is_superadmin;
