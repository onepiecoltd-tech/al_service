-- +goose Up
ALTER TABLE users
    ADD COLUMN presence   TEXT NOT NULL DEFAULT 'offline',
    ADD COLUMN status_msg TEXT NOT NULL DEFAULT '';

UPDATE users SET presence = 'online', status_msg = 'Sẵn sàng luyện ngữ ✦' WHERE email = 'minhanh@email.com';
UPDATE users SET presence = 'online', status_msg = 'Sẵn sàng luyện đề ✦'  WHERE email = 'thuha@email.com';
UPDATE users SET presence = 'busy',   status_msg = 'Đang thi đấu'         WHERE email = 'khanh@email.com';
UPDATE users SET presence = 'away',   status_msg = 'Ăn trưa, lát quay lại' WHERE email = 'nam@email.com';
UPDATE users SET presence = 'offline' WHERE email IN ('linh@email.com', 'phuc@email.com');

CREATE TABLE friendships (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id)
);

-- Seed minhanh's friendships (both directions).
INSERT INTO friendships (user_id, friend_id)
SELECT a.id, b.id FROM users a, users b
WHERE a.email = 'minhanh@email.com'
  AND b.email IN ('khanh@email.com', 'thuha@email.com', 'linh@email.com', 'nam@email.com', 'phuc@email.com');

INSERT INTO friendships (user_id, friend_id)
SELECT b.id, a.id FROM users a, users b
WHERE a.email = 'minhanh@email.com'
  AND b.email IN ('khanh@email.com', 'thuha@email.com', 'linh@email.com', 'nam@email.com', 'phuc@email.com');

-- +goose Down
DROP TABLE friendships;
ALTER TABLE users DROP COLUMN presence, DROP COLUMN status_msg;
