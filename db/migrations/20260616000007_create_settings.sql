-- +goose Up
CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    label      TEXT NOT NULL,
    value      BOOLEAN NOT NULL DEFAULT FALSE,
    sort       INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (key, label, value, sort) VALUES
('allow_signup',     'Cho phép đăng ký mới',        TRUE,  1),
('community_blog',   'Blog cộng đồng (cần duyệt)',  TRUE,  2),
('livestream_gifts', 'Livestream + tặng quà',       TRUE,  3),
('maintenance',      'Chế độ bảo trì',              FALSE, 4);

-- +goose Down
DROP TABLE settings;
