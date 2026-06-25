package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type QuestionRepository interface {
	// ReplaceForExam deletes the exam's existing questions and inserts the new
	// set in one transaction, so re-importing is idempotent.
	ReplaceForExam(ctx context.Context, examID uuid.UUID, qs []model.Question) error
	ListByExam(ctx context.Context, examID uuid.UUID) ([]model.Question, error)
	// RandomFromBank returns one random question from the published admin
	// question bank (owner_id IS NULL exams), for use as a speaking prompt.
	RandomFromBank(ctx context.Context) (*model.Question, error)
	// ListMissingAnswers returns up to limit questions that still have no sample
	// answer (and haven't exhausted answer attempts), for the backfill job. Backed
	// by a partial index so it never scans the full questions table.
	ListMissingAnswers(ctx context.Context, limit int) ([]model.QuestionNeedingAnswer, error)
	// SetSampleAnswer stores a generated answer for a question.
	SetSampleAnswer(ctx context.Context, id uuid.UUID, answer string) error
	// BumpAnswerAttempt records a failed answer-generation attempt, so a question
	// the AI can't answer eventually drops out of the backfill index.
	BumpAnswerAttempt(ctx context.Context, id uuid.UUID) error
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

func (r *questionRepository) RandomFromBank(ctx context.Context) (*model.Question, error) {
	var q model.Question
	err := r.db.QueryRow(ctx,
		`SELECT q.id, q.exam_id, q.position, q.prompt, q.sample_answer, q.created_at
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE e.owner_id IS NULL AND e.state = 'published'
		 ORDER BY random() LIMIT 1`).
		Scan(&q.ID, &q.ExamID, &q.Position, &q.Prompt, &q.SampleAnswer, &q.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("ngân hàng chưa có câu hỏi nào")
		}
		return nil, apperror.Internal(err)
	}
	return &q, nil
}

func (r *questionRepository) ListMissingAnswers(ctx context.Context, limit int) ([]model.QuestionNeedingAnswer, error) {
	// The WHERE matches idx_questions_missing_answer's predicate exactly so the
	// partial index is used — only unanswered, not-yet-exhausted rows are touched.
	rows, err := r.db.Query(ctx,
		`SELECT q.id, q.prompt, e.language
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE q.sample_answer = '' AND q.answer_attempts < 3
		 ORDER BY q.created_at
		 LIMIT $1`, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	out := []model.QuestionNeedingAnswer{}
	for rows.Next() {
		var q model.QuestionNeedingAnswer
		if err := rows.Scan(&q.ID, &q.Prompt, &q.Language); err != nil {
			return nil, apperror.Internal(err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return out, nil
}

func (r *questionRepository) SetSampleAnswer(ctx context.Context, id uuid.UUID, answer string) error {
	if _, err := r.db.Exec(ctx, `UPDATE questions SET sample_answer = $2 WHERE id = $1`, id, answer); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *questionRepository) BumpAnswerAttempt(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `UPDATE questions SET answer_attempts = answer_attempts + 1 WHERE id = $1`, id); err != nil {
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
