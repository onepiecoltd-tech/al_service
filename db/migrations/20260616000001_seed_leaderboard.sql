-- +goose Up
ALTER TABLE users ADD COLUMN wins INTEGER NOT NULL DEFAULT 0;

UPDATE users SET wins = 41 WHERE email = 'minhanh@email.com';

-- Extra users so the leaderboard has content. All share the dev password "password".
INSERT INTO users (email, display_name, password_hash, handle, plan, elo, wins, streak) VALUES
('khanh@email.com', 'Khánh', '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@khanh', 'Pro',  1521, 48, 9),
('thuha@email.com', 'Thu Hà', '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@thuha', 'Free', 1455, 39, 5),
('linh@email.com',  'Linh',  '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@linh',  'Free', 1402, 33, 3),
('nam@email.com',   'Nam',   '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@nam',   'Free', 1310, 25, 0),
('phuc@email.com',  'Phúc',  '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@phuc',  'Free', 1288, 21, 1);

-- +goose Down
DELETE FROM users WHERE email IN ('khanh@email.com', 'thuha@email.com', 'linh@email.com', 'nam@email.com', 'phuc@email.com');
ALTER TABLE users DROP COLUMN wins;
