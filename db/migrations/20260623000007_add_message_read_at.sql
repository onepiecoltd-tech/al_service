-- +goose Up
ALTER TABLE direct_messages ADD COLUMN read_at TIMESTAMPTZ;

-- Existing messages predate read tracking — treat them as already read so
-- they don't all suddenly count as unread.
UPDATE direct_messages SET read_at = created_at;

CREATE INDEX direct_messages_unread_idx ON direct_messages (receiver_id) WHERE read_at IS NULL;

-- +goose Down
DROP INDEX direct_messages_unread_idx;
ALTER TABLE direct_messages DROP COLUMN read_at;
