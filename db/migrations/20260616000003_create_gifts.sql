-- +goose Up
CREATE TABLE gifts (
    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    emoji TEXT NOT NULL,
    name  TEXT NOT NULL,
    price INTEGER NOT NULL
);

INSERT INTO gifts (emoji, name, price) VALUES
('🌹', 'Hồng', 10),
('👏', 'Vỗ tay', 15),
('❤️', 'Tim', 20),
('🔥', 'Lửa', 30),
('🎓', 'Tốt nghiệp', 60),
('🏆', 'Cúp', 120),
('💎', 'Kim cương', 500);

-- +goose Down
DROP TABLE gifts;
