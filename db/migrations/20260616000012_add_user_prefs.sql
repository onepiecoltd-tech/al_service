-- +goose Up
ALTER TABLE users ADD COLUMN prefs JSONB NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE users DROP COLUMN prefs;
