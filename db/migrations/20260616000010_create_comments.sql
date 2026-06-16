-- +goose Up
CREATE TABLE comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES blog_posts(id) ON DELETE CASCADE,
    author     TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_post ON comments (post_id, created_at);

-- Seeded posts had fake comment counts but no rows; reset so the count column
-- stays in sync with the real comments table going forward.
UPDATE blog_posts SET comments = 0;

-- +goose Down
DROP TABLE comments;
