-- +goose Up
CREATE TYPE student.import_job_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS student.import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name VARCHAR(255) NOT NULL,
    total_records INT NOT NULL DEFAULT 0,
    processed_records INT NOT NULL DEFAULT 0,
    successful_records INT NOT NULL DEFAULT 0,
    failed_records INT NOT NULL DEFAULT 0,
    status student.import_job_status DEFAULT 'pending',
    errors JSONB DEFAULT '[]',
    created_by UUID NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_import_jobs_status ON student.import_jobs(status);
CREATE INDEX idx_import_jobs_created_by ON student.import_jobs(created_by);
CREATE INDEX idx_import_jobs_created_at ON student.import_jobs(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS student.import_jobs;
DROP TYPE IF EXISTS student.import_job_status;
