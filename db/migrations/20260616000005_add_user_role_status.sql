-- +goose Up
ALTER TABLE users
    ADD COLUMN role   TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

UPDATE users SET role = 'admin' WHERE email = 'minhanh@email.com';
UPDATE users SET role = 'mod'   WHERE email = 'thuha@email.com';
UPDATE users SET status = 'banned' WHERE email = 'nam@email.com';

-- Extra user from the admin mock (login password: "password").
INSERT INTO users (email, display_name, password_hash, handle, plan, elo, role, status) VALUES
('quynh@email.com', 'Quỳnh', '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG', '@quynh', 'Pro', 1390, 'user', 'active');

-- +goose Down
DELETE FROM users WHERE email = 'quynh@email.com';
ALTER TABLE users DROP COLUMN role, DROP COLUMN status;
