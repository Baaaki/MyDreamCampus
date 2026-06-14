-- +goose Up
CREATE TABLE IF NOT EXISTS auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    refresh_token_jti VARCHAR(255) UNIQUE NOT NULL,
    device_info VARCHAR(255),
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_sessions_jti ON auth.sessions(refresh_token_jti);
CREATE INDEX idx_sessions_expires ON auth.sessions(expires_at);

-- +goose Down
DROP INDEX IF EXISTS auth.idx_sessions_expires;
DROP INDEX IF EXISTS auth.idx_sessions_jti;
DROP INDEX IF EXISTS auth.idx_sessions_user_id;
DROP TABLE IF EXISTS auth.sessions;
