-- +goose Up
ALTER TABLE users
    ADD COLUMN handle TEXT NOT NULL DEFAULT '',
    ADD COLUMN plan   TEXT NOT NULL DEFAULT 'Free',
    ADD COLUMN coins  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN elo    INTEGER NOT NULL DEFAULT 1000,
    ADD COLUMN streak INTEGER NOT NULL DEFAULT 0;

UPDATE users
SET handle = '@minhanh', plan = 'Pro', coins = 1240, elo = 1482, streak = 12
WHERE email = 'minhanh@email.com';

-- +goose Down
ALTER TABLE users
    DROP COLUMN handle,
    DROP COLUMN plan,
    DROP COLUMN coins,
    DROP COLUMN elo,
    DROP COLUMN streak;
