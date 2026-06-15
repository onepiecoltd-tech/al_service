package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type NotificationRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Notification, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type notificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Notification, error) {
	const query = `
		SELECT id, type, icon, text, tone, read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	items := []model.Notification{}
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Icon, &n.Text, &n.Tone, &n.Read, &n.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return items, nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}
