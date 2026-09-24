-- +goose Up
-- One row per checkout meal starts. Card data is limited to brand and last
-- four digits: the number and CVC are never stored anywhere.
CREATE TYPE payment.payment_status_enum AS ENUM ('pending', 'completed', 'failed', 'expired', 'refunded');

CREATE TABLE IF NOT EXISTS payment.payments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    -- meal's "res_<uuid>" or "bat_<uuid>"; unique so a retried initiate
    -- returns the existing payment instead of opening a second one.
    reference_id TEXT NOT NULL UNIQUE,
    student_id UUID NOT NULL,
    amount NUMERIC(10,2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status payment.payment_status_enum NOT NULL DEFAULT 'pending',
    card_brand TEXT,
    card_last4 CHAR(4),
    failure_reason TEXT,
    -- Service clock, like every business deadline: the time machine moves it.
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_pending_expiry ON payment.payments(expires_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS payment.payments;
DROP TYPE IF EXISTS payment.payment_status_enum;
