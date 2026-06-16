-- +goose Up
CREATE TABLE coin_packs (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vnd     INTEGER NOT NULL,
    coins   INTEGER NOT NULL,
    popular BOOLEAN NOT NULL DEFAULT FALSE,
    sort    INTEGER NOT NULL DEFAULT 0
);

INSERT INTO coin_packs (vnd, coins, popular, sort) VALUES
(20000, 230, FALSE, 1),
(50000, 600, TRUE, 2),
(100000, 1300, FALSE, 3);

CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,            -- topup | gift
    coins       INTEGER NOT NULL,         -- signed (+topup, -gift)
    vnd         INTEGER NOT NULL DEFAULT 0,
    method      TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'ok', -- ok | failed
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user ON transactions (user_id, created_at DESC);
CREATE INDEX idx_transactions_created ON transactions (created_at DESC);

-- Seed some topups so the revenue screen has data (this month).
INSERT INTO transactions (user_id, kind, coins, vnd, method, description, status, created_at)
SELECT u.id, 'topup', v.coins, v.vnd, 'PayOS', 'Nạp xu', v.status, now() - v.ago
FROM users u
JOIN (VALUES
  ('minhanh@email.com', 600,  50000,  'ok',     INTERVAL '2 hours'),
  ('khanh@email.com',   1300, 100000, 'ok',     INTERVAL '5 hours'),
  ('quynh@email.com',   230,  20000,  'ok',     INTERVAL '1 day'),
  ('thuha@email.com',   600,  50000,  'ok',     INTERVAL '2 days'),
  ('nam@email.com',     230,  20000,  'failed', INTERVAL '6 hours')
) AS v(email, coins, vnd, status, ago) ON u.email = v.email;

-- +goose Down
DROP TABLE transactions;
DROP TABLE coin_packs;
