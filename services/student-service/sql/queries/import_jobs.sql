-- name: CreateImportJob :one
INSERT INTO import_jobs (file_name, total_records, created_by)
VALUES ($1, $2, $3)
RETURNING id, file_name, total_records, processed_records, successful_records,
          failed_records, status, errors, created_by, started_at, completed_at, created_at;

-- name: GetImportJobByID :one
SELECT id, file_name, total_records, processed_records, successful_records,
       failed_records, status, errors, created_by, started_at, completed_at, created_at
FROM import_jobs
WHERE id = $1
LIMIT 1;

-- name: UpdateImportJobProgress :exec
UPDATE import_jobs
SET processed_records = $2,
    successful_records = $3,
    failed_records = $4,
    errors = $5
WHERE id = $1;

-- name: StartImportJob :exec
UPDATE import_jobs
SET status = 'processing', started_at = NOW()
WHERE id = $1;

-- name: CompleteImportJob :exec
UPDATE import_jobs
SET status = 'completed', completed_at = NOW()
WHERE id = $1;

-- name: FailImportJob :exec
UPDATE import_jobs
SET status = 'failed', completed_at = NOW()
WHERE id = $1;

-- name: ListImportJobsByUser :many
SELECT id, file_name, total_records, processed_records, successful_records,
       failed_records, status, errors, created_by, started_at, completed_at, created_at
FROM import_jobs
WHERE created_by = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountImportJobsByUser :one
SELECT COUNT(*) FROM import_jobs WHERE created_by = $1;
