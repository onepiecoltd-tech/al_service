package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type QuestionRepository interface {
	// ReplaceForExam deletes the exam's existing questions and inserts the new
	// set in one transaction, so re-importing is idempotent.
	ReplaceForExam(ctx context.Context, examID uuid.UUID, qs []model.Question) error
	ListByExam(ctx context.Context, examID uuid.UUID) ([]model.Question, error)
}

type questionRepository struct {
	db *pgxpool.Pool
}

func NewQuestionRepository(db *pgxpool.Pool) QuestionRepository {
	return &questionRepository{db: db}
}

func (r *questionRepository) ReplaceForExam(ctx context.Context, examID uuid.UUID, qs []model.Question) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx, `DELETE FROM questions WHERE exam_id = $1`, examID); err != nil {
		return apperror.Internal(err)
	}
	for _, q := range qs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO questions (exam_id, position, prompt, sample_answer) VALUES ($1, $2, $3, $4)`,
			examID, q.Position, q.Prompt, q.SampleAnswer); err != nil {
			return apperror.Internal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *questionRepository) ListByExam(ctx context.Context, examID uuid.UUID) ([]model.Question, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, exam_id, position, prompt, sample_answer, created_at
		 FROM questions WHERE exam_id = $1 ORDER BY position`, examID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	qs := []model.Question{}
	for rows.Next() {
		var q model.Question
		if err := rows.Scan(&q.ID, &q.ExamID, &q.Position, &q.Prompt, &q.SampleAnswer, &q.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return qs, nil
}
