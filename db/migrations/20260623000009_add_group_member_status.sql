-- +goose Up
-- Existing memberships are already-established, so default to 'approved';
-- new join-by-code requests insert as 'pending' until the owner approves.
ALTER TABLE study_group_members
    ADD COLUMN status TEXT NOT NULL DEFAULT 'approved' CHECK (status IN ('pending', 'approved'));

-- +goose Down
ALTER TABLE study_group_members DROP COLUMN status;
