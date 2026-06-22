-- +goose Up
-- Exams uploaded by a normal user are owned by them (NULL = admin/global bank).
ALTER TABLE exams ADD COLUMN owner_id UUID REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX idx_exams_owner_id ON exams(owner_id);

-- +goose Down
DROP INDEX idx_exams_owner_id;
ALTER TABLE exams DROP COLUMN owner_id;
