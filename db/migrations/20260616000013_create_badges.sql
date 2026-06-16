-- +goose Up
CREATE TABLE badges (
    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    emoji TEXT NOT NULL,
    name  TEXT NOT NULL,
    tone  TEXT NOT NULL DEFAULT 'son',
    sort  INTEGER NOT NULL DEFAULT 0
);

INSERT INTO badges (emoji, name, tone, sort) VALUES
('🔥', 'Streak 7+', 'son', 1),
('🏆', 'Top 20 tuần', 'gold', 2),
('🎤', 'Speaking 6.5', 'reu', 3),
('⚔️', '40 trận thắng', 'son', 4),
('📚', '300 lượt luyện', 'reu', 5),
('💎', 'Nhà hảo tâm', 'gold', 6),
('🎓', 'Hoàn thành 10 đề', 'reu', 7),
('⭐', 'Tân binh', 'ink', 8);

CREATE TABLE user_badges (
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id  UUID NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, badge_id)
);

-- Grant all badges to the demo account so the profile shows them.
INSERT INTO user_badges (user_id, badge_id)
SELECT u.id, b.id FROM users u, badges b WHERE u.email = 'minhanh@email.com';

-- +goose Down
DROP TABLE user_badges;
DROP TABLE badges;
