-- +goose Up
CREATE TABLE reports (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content    TEXT NOT NULL,
    reporter   TEXT NOT NULL,
    type       TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'warn', -- err | warn
    status     TEXT NOT NULL DEFAULT 'open', -- open | resolved
    action     TEXT NOT NULL DEFAULT '',     -- dismissed | hidden | removed
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reports_status ON reports (status, created_at DESC);

INSERT INTO reports (content, reporter, type, severity, created_at) VALUES
('Bình luận xúc phạm trong bài "TOEIC 900+"', 'Nam',     'Bình luận',   'err',  now() - INTERVAL '12 minutes'),
('Spam link trong chat phòng nhóm',           'ẩn danh', 'Chat',        'warn', now() - INTERVAL '1 hour'),
('Nội dung không phù hợp trong livestream',    'Phúc',    'Livestream',  'err',  now() - INTERVAL '3 hours'),
('Bài blog nghi sao chép',                     'Quỳnh',   'Blog',        'warn', now() - INTERVAL '1 day');

-- +goose Down
DROP TABLE reports;
