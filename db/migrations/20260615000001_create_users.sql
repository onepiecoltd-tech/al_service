-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed account for local development (password: "password").
INSERT INTO users (email, display_name, password_hash) VALUES (
    'minhanh@email.com',
    'Minh Anh',
    '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG'
);

-- +goose Down
DROP TABLE users;
