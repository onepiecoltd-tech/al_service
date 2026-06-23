package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type ChatMessageRepository interface {
	ListByExam(ctx context.Context, examID uuid.UUID) ([]model.ChatMessage, error)
	Insert(ctx context.Context, examID uuid.UUID, role, text string) error
}

type chatMessageRepository struct {
	db *pgxpool.Pool
}

func NewChatMessageRepository(db *pgxpool.Pool) ChatMessageRepository {
	return &chatMessageRepository{db: db}
}

func (r *chatMessageRepository) ListByExam(ctx context.Context, examID uuid.UUID) ([]model.ChatMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, exam_id, role, text, created_at
		 FROM exam_chat_messages WHERE exam_id = $1 ORDER BY created_at`, examID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	msgs := []model.ChatMessage{}
	for rows.Next() {
		var m model.ChatMessage
		if err := rows.Scan(&m.ID, &m.ExamID, &m.Role, &m.Text, &m.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return msgs, nil
}

func (r *chatMessageRepository) Insert(ctx context.Context, examID uuid.UUID, role, text string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO exam_chat_messages (exam_id, role, text) VALUES ($1, $2, $3)`,
		examID, role, text)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}
