-- +goose Up
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    icon       TEXT NOT NULL,
    text       TEXT NOT NULL,
    tone       TEXT NOT NULL DEFAULT 'son',
    read       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications (user_id, created_at DESC);

-- Seed notifications for minhanh.
INSERT INTO notifications (user_id, type, icon, text, tone, created_at)
SELECT u.id, v.type, v.icon, v.text, v.tone, now() - v.ago
FROM users u,
(VALUES
  ('invite', 'swords',    'Khánh mời bạn vào một trận thách đấu',        'son',   INTERVAL '2 minutes'),
  ('go_live', 'radio',    'Thu Hà đang livestream một trận đấu',         'error', INTERVAL '15 minutes'),
  ('gift', 'gift',        'Bạn nhận 🏆 từ Nam trong buổi live',          'gold',  INTERVAL '1 hour'),
  ('friend', 'user-plus', 'Linh đã chấp nhận lời mời kết bạn',           'reu',   INTERVAL '1 day')
) AS v(type, icon, text, tone, ago)
WHERE u.email = 'minhanh@email.com';

-- +goose Down
DROP TABLE notifications;
