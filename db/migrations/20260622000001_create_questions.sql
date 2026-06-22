-- +goose Up
CREATE TABLE questions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id       UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    prompt        TEXT NOT NULL,
    sample_answer TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_questions_exam ON questions (exam_id, position);

-- +goose Down
DROP TABLE questions;
