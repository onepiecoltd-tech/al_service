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

INSERT INTO exams (name, type, questions, author, state, created_at) VALUES
('Cambridge IELTS 18 — Full', 'IELTS',   200, 'Admin',  'published', now() - INTERVAL '14 days'),
('TOEIC ETS 2024 — Test 5',   'TOEIC',   200, 'Admin',  'published', now() - INTERVAL '19 days'),
('Academic Word List drills', 'Từ vựng', 120, 'Thu Hà', 'review',    now()),
('TOEFL Reading set A',        'TOEFL',   60,  'Khánh',  'review',    now() - INTERVAL '1 day'),
('IELTS Speaking forecast Q2', 'IELTS',   45,  'Admin',  'draft',     now() - INTERVAL '17 days');

-- +goose Down
DROP TABLE exams;
