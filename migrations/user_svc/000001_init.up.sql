CREATE SCHEMA IF NOT EXISTS user_svc;

CREATE TABLE user_svc.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(32) NOT NULL,
    email VARCHAR(320) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(64) NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_username ON user_svc.users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email ON user_svc.users(email) WHERE deleted_at IS NULL;
