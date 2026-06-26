-- +goose Up
-- Per-question skill (listening/reading/writing/speaking), classified by the AI
-- during extraction. Empty until set.
ALTER TABLE questions ADD COLUMN type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE questions DROP COLUMN type;
