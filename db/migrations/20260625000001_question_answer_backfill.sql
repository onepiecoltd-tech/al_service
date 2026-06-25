-- +goose Up
-- Track how many times the AI answer-backfill job has tried a question, so a
-- persistently unanswerable one stops being retried and doesn't clog the batch.
ALTER TABLE questions ADD COLUMN answer_attempts INT NOT NULL DEFAULT 0;

-- Partial index: the backfill job only ever looks at questions still missing an
-- answer and not yet exhausted on attempts, so it never scans the full table.
-- The predicate matches the job's query exactly (sample_answer = '' AND
-- answer_attempts < 3), and indexing created_at gives a cheap ordered scan.
CREATE INDEX idx_questions_missing_answer ON questions (created_at)
    WHERE sample_answer = '' AND answer_attempts < 3;

-- +goose Down
DROP INDEX idx_questions_missing_answer;
ALTER TABLE questions DROP COLUMN answer_attempts;
