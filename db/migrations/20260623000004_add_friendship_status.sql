-- +goose Up
-- Existing rows are already-established (seeded) friendships — keep them
-- accepted. New requests via the API will insert as 'pending' until the
-- other side confirms.
ALTER TABLE friendships ADD COLUMN status TEXT NOT NULL DEFAULT 'accepted' CHECK (status IN ('pending', 'accepted'));

-- +goose Down
ALTER TABLE friendships DROP COLUMN status;
