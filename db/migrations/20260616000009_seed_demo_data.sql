-- +goose Up
-- Demo rows to exercise pagination. Reversible in Down.

INSERT INTO blog_posts (title, category, author, excerpt, reads, comments, status, created_at)
SELECT
  'Bài viết mẫu #' || g,
  (ARRAY['IELTS', 'TOEIC', 'TOEFL', 'Phương pháp', 'Câu chuyện'])[1 + (g % 5)],
  (ARRAY['Minh Anh', 'Thu Hà', 'Khánh', 'Linh', 'Nam'])[1 + (g % 5)],
  'Tóm tắt nội dung mẫu cho bài viết số ' || g || '.',
  (g * 37) % 3000,
  g % 25,
  (ARRAY['published', 'published', 'review', 'draft'])[1 + (g % 4)],
  now() - (g || ' hours')::interval
FROM generate_series(1, 30) AS g;

INSERT INTO exams (name, type, questions, author, state, created_at)
SELECT
  (ARRAY['IELTS', 'TOEIC', 'TOEFL', 'Từ vựng'])[1 + (g % 4)] || ' — bộ đề #' || g,
  (ARRAY['IELTS', 'TOEIC', 'TOEFL', 'Từ vựng'])[1 + (g % 4)],
  20 + (g * 7) % 200,
  (ARRAY['Admin', 'Thu Hà', 'Khánh'])[1 + (g % 3)],
  (ARRAY['published', 'published', 'review', 'draft'])[1 + (g % 4)],
  now() - (g || ' hours')::interval
FROM generate_series(1, 30) AS g;

INSERT INTO reports (content, reporter, type, severity, status, created_at)
SELECT
  'Báo cáo mẫu #' || g || ' — nội dung cần xem xét',
  (ARRAY['Nam', 'Phúc', 'ẩn danh', 'Quỳnh'])[1 + (g % 4)],
  (ARRAY['Bình luận', 'Chat', 'Livestream', 'Blog'])[1 + (g % 4)],
  (ARRAY['err', 'warn'])[1 + (g % 2)],
  'open',
  now() - (g || ' minutes')::interval
FROM generate_series(1, 30) AS g;

-- All share the dev password "password".
INSERT INTO users (email, display_name, password_hash, handle, plan, elo, wins, streak, role, status)
SELECT
  'demo' || g || '@email.com',
  'Học viên ' || g,
  '$2a$10$Ln5iBGuXq5cs3BwXx.eUBOUOmCPJHtuB62B5HWgIU9dBT627QqgPG',
  '@demo' || g,
  (ARRAY['Free', 'Pro'])[1 + (g % 2)],
  1000 + (g * 13) % 600,
  (g * 3) % 50,
  g % 30,
  'user',
  (ARRAY['active', 'active', 'active', 'banned'])[1 + (g % 4)]
FROM generate_series(1, 30) AS g;

-- +goose Down
DELETE FROM blog_posts WHERE title LIKE 'Bài viết mẫu #%';
DELETE FROM exams WHERE name LIKE '% — bộ đề #%';
DELETE FROM reports WHERE content LIKE 'Báo cáo mẫu #%';
DELETE FROM users WHERE email LIKE 'demo%@email.com';
