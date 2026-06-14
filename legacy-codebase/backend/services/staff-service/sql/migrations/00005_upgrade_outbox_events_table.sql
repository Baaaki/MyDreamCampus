-- +goose Up
-- Upgrade outbox_events to match student-service pattern with status enum and retry support

-- Create status enum
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

-- Add new columns
ALTER TABLE outbox_events
    ADD COLUMN status outbox_status_enum DEFAULT 'pending',
    ADD COLUMN retry_count SMALLINT DEFAULT 0,
    ADD COLUMN max_retries SMALLINT DEFAULT 3,
    ADD COLUMN error_message TEXT;

-- Migrate existing data: processed=true -> 'processed', processed=false -> 'pending'
UPDATE outbox_events SET status = 'processed' WHERE processed = true;
UPDATE outbox_events SET status = 'pending' WHERE processed = false;

-- Drop old column and indexes
DROP INDEX IF EXISTS idx_outbox_events_processed;
DROP INDEX IF EXISTS idx_outbox_events_created_at;
ALTER TABLE outbox_events DROP COLUMN processed;

-- Change id from SERIAL to UUID
ALTER TABLE outbox_events
    ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
UPDATE outbox_events SET new_id = gen_random_uuid();
ALTER TABLE outbox_events DROP CONSTRAINT outbox_events_pkey;
ALTER TABLE outbox_events DROP COLUMN id;
ALTER TABLE outbox_events RENAME COLUMN new_id TO id;
ALTER TABLE outbox_events ADD PRIMARY KEY (id);

-- Create new indexes matching student-service
CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_events_retry ON outbox_events(status, retry_count) WHERE status = 'failed';

-- +goose Down
-- Revert to simple boolean processed column

DROP INDEX IF EXISTS idx_outbox_events_pending;
DROP INDEX IF EXISTS idx_outbox_events_retry;

-- Add back old columns
ALTER TABLE outbox_events
    ADD COLUMN processed BOOLEAN NOT NULL DEFAULT false;

-- Migrate data back
UPDATE outbox_events SET processed = true WHERE status = 'processed';
UPDATE outbox_events SET processed = false WHERE status != 'processed';

-- Remove new columns
ALTER TABLE outbox_events
    DROP COLUMN status,
    DROP COLUMN retry_count,
    DROP COLUMN max_retries,
    DROP COLUMN error_message;

-- Revert id to SERIAL
ALTER TABLE outbox_events DROP CONSTRAINT outbox_events_pkey;
ALTER TABLE outbox_events DROP COLUMN id;
ALTER TABLE outbox_events ADD COLUMN id SERIAL PRIMARY KEY;

DROP TYPE IF EXISTS outbox_status_enum;

-- Recreate old indexes
CREATE INDEX idx_outbox_events_processed ON outbox_events(processed);
CREATE INDEX idx_outbox_events_created_at ON outbox_events(created_at);
