package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type DirectMessageRepository interface {
	// ListConversation returns the messages between the two users, oldest first.
	ListConversation(ctx context.Context, userA, userB uuid.UUID, limit int) ([]model.DirectMessage, error)
	Insert(ctx context.Context, senderID, receiverID uuid.UUID, body string) (*model.DirectMessage, error)
}

type directMessageRepository struct {
	db *pgxpool.Pool
}

func NewDirectMessageRepository(db *pgxpool.Pool) DirectMessageRepository {
	return &directMessageRepository{db: db}
}

func (r *directMessageRepository) ListConversation(ctx context.Context, userA, userB uuid.UUID, limit int) ([]model.DirectMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, sender_id, receiver_id, body, created_at
		 FROM direct_messages
		 WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
		 ORDER BY created_at DESC LIMIT $3`,
		userA, userB, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	msgs := []model.DirectMessage{}
	for rows.Next() {
		var m model.DirectMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Body, &m.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	// reverse to chronological order (oldest first)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (r *directMessageRepository) Insert(ctx context.Context, senderID, receiverID uuid.UUID, body string) (*model.DirectMessage, error) {
	var m model.DirectMessage
	err := r.db.QueryRow(ctx,
		`INSERT INTO direct_messages (sender_id, receiver_id, body) VALUES ($1, $2, $3)
		 RETURNING id, sender_id, receiver_id, body, created_at`,
		senderID, receiverID, body).
		Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Body, &m.CreatedAt)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &m, nil
}
