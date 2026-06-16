package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type OverviewRepository interface {
	Get(ctx context.Context) (*model.Overview, error)
}

type overviewRepository struct {
	db *pgxpool.Pool
}

func NewOverviewRepository(db *pgxpool.Pool) OverviewRepository {
	return &overviewRepository{db: db}
}

func (r *overviewRepository) Get(ctx context.Context) (*model.Overview, error) {
	var o model.Overview
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM users WHERE plan = 'Pro'),
			(SELECT count(*) FROM reports WHERE status = 'open'),
			(SELECT count(*) FROM exams WHERE state = 'review'),
			(SELECT count(*) FROM blog_posts)
	`).Scan(&o.UsersTotal, &o.ProTotal, &o.ReportsOpen, &o.ExamsReview, &o.PostsTotal)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, display_name, email, plan, created_at FROM users ORDER BY created_at DESC LIMIT 5`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	o.RecentUsers = []model.RecentUser{}
	for rows.Next() {
		var u model.RecentUser
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Plan, &u.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		o.RecentUsers = append(o.RecentUsers, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return &o, nil
}
