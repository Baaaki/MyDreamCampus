-- name: CreateProcessedEvent :exec
INSERT INTO student.processed_events (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;

-- name: IsEventProcessed :one
SELECT EXISTS(SELECT 1 FROM student.processed_events WHERE event_id = $1);

-- name: DeleteOldProcessedEvents :exec
DELETE FROM student.processed_events WHERE processed_at < NOW() - INTERVAL '30 days';
