-- +goose Up
ALTER TABLE closed_days ADD COLUMN semester VARCHAR(50);

-- +goose Down
ALTER TABLE closed_days DROP COLUMN semester;
