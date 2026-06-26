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
	// ListBySkill pools up to limit random questions of a given skill across the
	// published bank and the user's own exams, optionally filtered by language.
	// source narrows to "bank" or "mine" ("" = both); search filters by a
	// case-insensitive substring over the prompt and sample answer ("" = no filter).
	ListBySkill(ctx context.Context, userID uuid.UUID, skill, lang, source, search string, limit int) ([]model.Question, error)
	// ListAdmin returns a paginated, filtered slice of questions (joined with their
	// exam's name and language) for the admin management screen. skill and lang are
	// optional equality filters ("" = any); answered is "yes"/"no"/"" (has a sample
	// answer or not); search is a case-insensitive substring over prompt and answer.
	ListAdmin(ctx context.Context, skill, lang, answered, search string, limit, offset int) ([]model.AdminQuestion, int, error)
	// GetAdmin returns one question enriched with its exam's name and language,
	// for the admin question detail/edit page.
	GetAdmin(ctx context.Context, id uuid.UUID) (model.AdminQuestion, error)
	// UpdateContent edits a question's prompt and sample answer (admin).
	UpdateContent(ctx context.Context, id uuid.UUID, prompt, sampleAnswer string) error
	// GetForAnswer returns one question's prompt and its exam's language, for
	// generating a sample answer on demand.
	GetForAnswer(ctx context.Context, id uuid.UUID) (model.QuestionNeedingAnswer, error)
	// Delete removes a single question (admin).
	Delete(ctx context.Context, id uuid.UUID) error
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
			`INSERT INTO questions (exam_id, position, prompt, sample_answer, type) VALUES ($1, $2, $3, $4, $5)`,
			examID, q.Position, q.Prompt, q.SampleAnswer, q.Type); err != nil {
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
		`SELECT q.id, q.exam_id, q.position, q.prompt, q.sample_answer, q.type, q.created_at
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE e.owner_id IS NULL AND e.state = 'published'
		 ORDER BY random() LIMIT 1`).
		Scan(&q.ID, &q.ExamID, &q.Position, &q.Prompt, &q.SampleAnswer, &q.Type, &q.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("ngân hàng chưa có câu hỏi nào")
		}
		return nil, apperror.Internal(err)
	}
	return &q, nil
}

func (r *questionRepository) ListAdmin(ctx context.Context, skill, lang, answered, search string, limit, offset int) ([]model.AdminQuestion, int, error) {
	// Shared WHERE so the count and page queries stay in sync. Empty params
	// disable their filter; answered = 'yes'/'no' checks for a non-empty answer.
	const filter = `($1 = '' OR q.type = $1)
		AND ($2 = '' OR e.language = $2)
		AND ($3 = '' OR ($3 = 'yes' AND q.sample_answer <> '') OR ($3 = 'no' AND q.sample_answer = ''))
		AND ($4 = '' OR q.prompt ILIKE '%' || $4 || '%' OR q.sample_answer ILIKE '%' || $4 || '%')`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM questions q JOIN exams e ON e.id = q.exam_id WHERE `+filter,
		skill, lang, answered, search).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT q.id, q.exam_id, e.name, e.language, q.position, q.prompt, q.sample_answer, q.type, q.created_at
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE `+filter+`
		 ORDER BY q.created_at DESC, q.position
		 LIMIT $5 OFFSET $6`, skill, lang, answered, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	qs := []model.AdminQuestion{}
	for rows.Next() {
		var q model.AdminQuestion
		if err := rows.Scan(&q.ID, &q.ExamID, &q.ExamName, &q.Language, &q.Position, &q.Prompt, &q.SampleAnswer, &q.Type, &q.CreatedAt); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return qs, total, nil
}

func (r *questionRepository) GetAdmin(ctx context.Context, id uuid.UUID) (model.AdminQuestion, error) {
	var q model.AdminQuestion
	err := r.db.QueryRow(ctx,
		`SELECT q.id, q.exam_id, e.name, e.language, q.position, q.prompt, q.sample_answer, q.type, q.created_at
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE q.id = $1`, id).
		Scan(&q.ID, &q.ExamID, &q.ExamName, &q.Language, &q.Position, &q.Prompt, &q.SampleAnswer, &q.Type, &q.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return q, apperror.NotFound("không tìm thấy câu hỏi")
		}
		return q, apperror.Internal(err)
	}
	return q, nil
}

func (r *questionRepository) UpdateContent(ctx context.Context, id uuid.UUID, prompt, sampleAnswer string) error {
	tag, err := r.db.Exec(ctx, `UPDATE questions SET prompt = $2, sample_answer = $3 WHERE id = $1`, id, prompt, sampleAnswer)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("không tìm thấy câu hỏi")
	}
	return nil
}

func (r *questionRepository) GetForAnswer(ctx context.Context, id uuid.UUID) (model.QuestionNeedingAnswer, error) {
	var q model.QuestionNeedingAnswer
	err := r.db.QueryRow(ctx,
		`SELECT q.id, q.prompt, e.language
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE q.id = $1`, id).Scan(&q.ID, &q.Prompt, &q.Language)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return q, apperror.NotFound("không tìm thấy câu hỏi")
		}
		return q, apperror.Internal(err)
	}
	return q, nil
}

func (r *questionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM questions WHERE id = $1`, id)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("không tìm thấy câu hỏi")
	}
	return nil
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

func (r *questionRepository) ListBySkill(ctx context.Context, userID uuid.UUID, skill, lang, source, search string, limit int) ([]model.Question, error) {
	rows, err := r.db.Query(ctx,
		`SELECT q.id, q.exam_id, q.position, q.prompt, q.sample_answer, q.type, q.created_at
		 FROM questions q JOIN exams e ON e.id = q.exam_id
		 WHERE q.type = $1
		   AND ($2 = '' OR e.language = $2)
		   AND (CASE $5
		          WHEN 'bank' THEN e.owner_id IS NULL AND e.state = 'published'
		          WHEN 'mine' THEN e.owner_id = $3
		          ELSE (e.owner_id IS NULL AND e.state = 'published') OR e.owner_id = $3
		        END)
		   AND ($6 = '' OR q.prompt ILIKE '%' || $6 || '%' OR q.sample_answer ILIKE '%' || $6 || '%')
		 ORDER BY random()
		 LIMIT $4`, skill, lang, userID, limit, source, search)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	qs := []model.Question{}
	for rows.Next() {
		var q model.Question
		if err := rows.Scan(&q.ID, &q.ExamID, &q.Position, &q.Prompt, &q.SampleAnswer, &q.Type, &q.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return qs, nil
}

func (r *questionRepository) ListByExam(ctx context.Context, examID uuid.UUID) ([]model.Question, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, exam_id, position, prompt, sample_answer, type, created_at
		 FROM questions WHERE exam_id = $1 ORDER BY position`, examID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	qs := []model.Question{}
	for rows.Next() {
		var q model.Question
		if err := rows.Scan(&q.ID, &q.ExamID, &q.Position, &q.Prompt, &q.SampleAnswer, &q.Type, &q.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return qs, nil
}
