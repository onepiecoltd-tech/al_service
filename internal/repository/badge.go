package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type BadgeRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Badge, error)
}

type badgeRepository struct {
	db *pgxpool.Pool
}

func NewBadgeRepository(db *pgxpool.Pool) BadgeRepository {
	return &badgeRepository{db: db}
}

func (r *badgeRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Badge, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.emoji, b.name, b.tone
		FROM user_badges ub JOIN badges b ON b.id = ub.badge_id
		WHERE ub.user_id = $1
		ORDER BY b.sort`, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	badges := []model.Badge{}
	for rows.Next() {
		var b model.Badge
		if err := rows.Scan(&b.ID, &b.Emoji, &b.Name, &b.Tone); err != nil {
			return nil, apperror.Internal(err)
		}
		badges = append(badges, b)
	}
	return badges, rows.Err()
}
