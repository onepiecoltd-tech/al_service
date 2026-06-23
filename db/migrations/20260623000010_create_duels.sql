-- +goose Up
CREATE TABLE duels (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenger_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    opponent_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prompt           TEXT NOT NULL,
    challenger_score INT  NOT NULL,
    opponent_score   INT,
    status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'declined')),
    winner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    challenger_delta INT  NOT NULL DEFAULT 0,
    opponent_delta   INT  NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ
);

CREATE INDEX duels_opponent_pending_idx ON duels (opponent_id) WHERE status = 'pending';
CREATE INDEX duels_participants_idx ON duels (challenger_id, opponent_id, created_at);

-- +goose Down
DROP TABLE duels;
