-- +goose Up
-- Persisted Giải đề AI conversation, scoped to the exam (which is already
-- owner-scoped, so no separate user_id column is needed here).
CREATE TABLE exam_chat_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id    UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('user', 'model')),
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exam_chat_messages_exam_id ON exam_chat_messages(exam_id, created_at);

-- +goose Down
DROP TABLE exam_chat_messages;
