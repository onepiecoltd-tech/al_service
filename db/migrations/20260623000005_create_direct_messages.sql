-- +goose Up
CREATE TABLE direct_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX direct_messages_conversation_idx ON direct_messages (
    LEAST(sender_id, receiver_id), GREATEST(sender_id, receiver_id), created_at
);

-- +goose Down
DROP TABLE direct_messages;
