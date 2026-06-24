-- +goose Up
-- The language the user is currently learning; drives the default exam-language
-- filter and upload language. Defaults to English for existing users.
ALTER TABLE users ADD COLUMN learning_language TEXT NOT NULL DEFAULT 'en';

-- +goose Down
ALTER TABLE users DROP COLUMN learning_language;
