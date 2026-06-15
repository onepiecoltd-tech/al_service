package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type SettingRepository interface {
	List(ctx context.Context) ([]model.Setting, error)
	Update(ctx context.Context, key string, value bool) (*model.Setting, error)
}

type settingRepository struct {
	db *pgxpool.Pool
}

func NewSettingRepository(db *pgxpool.Pool) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) List(ctx context.Context) ([]model.Setting, error) {
	rows, err := r.db.Query(ctx, `SELECT key, label, value, updated_at FROM settings ORDER BY sort`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	settings := []model.Setting{}
	for rows.Next() {
		var s model.Setting
		if err := rows.Scan(&s.Key, &s.Label, &s.Value, &s.UpdatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		settings = append(settings, s)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return settings, nil
}

func (r *settingRepository) Update(ctx context.Context, key string, value bool) (*model.Setting, error) {
	const query = `
		UPDATE settings SET value = $2, updated_at = now()
		WHERE key = $1
		RETURNING key, label, value, updated_at`

	var s model.Setting
	err := r.db.QueryRow(ctx, query, key, value).Scan(&s.Key, &s.Label, &s.Value, &s.UpdatedAt)
	if err != nil {
		return nil, apperror.NotFound("setting not found")
	}
	return &s, nil
}
