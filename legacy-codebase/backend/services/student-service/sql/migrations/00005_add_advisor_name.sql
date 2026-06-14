-- +goose Up
ALTER TABLE students ADD COLUMN advisor_name VARCHAR(200);

-- +goose Down
ALTER TABLE students DROP COLUMN advisor_name;
