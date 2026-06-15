package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type GiftRepository interface {
	List(ctx context.Context) ([]model.Gift, error)
}

type giftRepository struct {
	db *pgxpool.Pool
}

func NewGiftRepository(db *pgxpool.Pool) GiftRepository {
	return &giftRepository{db: db}
}

func (r *giftRepository) List(ctx context.Context) ([]model.Gift, error) {
	rows, err := r.db.Query(ctx, `SELECT id, emoji, name, price FROM gifts ORDER BY price`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	gifts := []model.Gift{}
	for rows.Next() {
		var g model.Gift
		if err := rows.Scan(&g.ID, &g.Emoji, &g.Name, &g.Price); err != nil {
			return nil, apperror.Internal(err)
		}
		gifts = append(gifts, g)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return gifts, nil
}
