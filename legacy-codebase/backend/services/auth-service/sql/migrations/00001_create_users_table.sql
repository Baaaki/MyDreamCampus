-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    department VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    token_version INT DEFAULT 1,
    force_password_change BOOLEAN DEFAULT TRUE,
    failed_login_attempts INT DEFAULT 0,
    locked_until TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Unique constraint only for active users (soft delete support)
CREATE UNIQUE INDEX idx_users_email_unique
    ON users(email) WHERE is_active = true;

CREATE INDEX idx_users_role ON users(role) WHERE is_active = true;
CREATE INDEX idx_users_department ON users(department) WHERE is_active = true;
CREATE INDEX idx_users_is_active ON users(is_active);

-- +goose Down
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_department;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email_unique;
DROP TABLE IF EXISTS users;
