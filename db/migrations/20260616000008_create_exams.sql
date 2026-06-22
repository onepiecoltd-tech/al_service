-- +goose Up
CREATE TABLE exams (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    questions  INTEGER NOT NULL DEFAULT 0,
    author     TEXT NOT NULL DEFAULT '',
    state      TEXT NOT NULL DEFAULT 'draft', -- published | review | draft
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE exams;
