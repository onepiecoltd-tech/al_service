-- +goose Up
CREATE TABLE blog_posts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    category   TEXT NOT NULL,
    author     TEXT NOT NULL,
    excerpt    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL DEFAULT '',
    reads      INTEGER NOT NULL DEFAULT 0,
    comments   INTEGER NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO blog_posts (title, category, author, excerpt, reads, comments, status, created_at) VALUES
('Chiến thuật Part 2 IELTS: kể chuyện theo khung 4 ý', 'IELTS Speaking', 'Thu Hà',
 'Cách biến 4 gạch đầu dòng của cue card thành một câu chuyện 2 phút mạch lạc, có mở–thân–kết và từ nối tự nhiên.',
 1200, 14, 'published', '2026-06-08 09:00:00+07'),
('Vì sao luyện đề từ chính tài liệu của bạn hiệu quả hơn', 'Phương pháp', 'Minh Anh',
 'Tải nguồn đề của riêng bạn, để AI sinh câu hỏi sát ngữ cảnh — ghi nhớ theo bối cảnh thật thay vì học vẹt.',
 860, 9, 'published', '2026-06-06 09:00:00+07'),
('TOEIC 900+: lịch luyện 6 tuần kèm mốc ELO thách đấu', 'TOEIC', 'Khánh',
 'Lộ trình kết hợp làm đề ngân hàng, thách đấu người lạ để giữ nhịp, và đo tiến bộ qua ELO mỗi tuần.',
 2400, 23, 'review', '2026-06-02 09:00:00+07');

-- +goose Down
DROP TABLE blog_posts;
