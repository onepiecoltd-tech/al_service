-- +goose Up
-- The target language an exam is for. Existing exams predate multi-language
-- support and are all English, so backfill them as 'en'.
ALTER TABLE exams ADD COLUMN language TEXT NOT NULL DEFAULT 'en';
CREATE INDEX idx_exams_language ON exams(language);

-- +goose Down
DROP INDEX idx_exams_language;
ALTER TABLE exams DROP COLUMN language;
