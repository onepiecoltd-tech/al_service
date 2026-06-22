package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type ExamRepository interface {
	List(ctx context.Context, limit, offset int) ([]model.Exam, int, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.Exam, int, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Exam, error)
	Create(ctx context.Context, e *model.Exam) error
	Update(ctx context.Context, e *model.Exam) (*model.Exam, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type examRepository struct {
	db *pgxpool.Pool
}

func NewExamRepository(db *pgxpool.Pool) ExamRepository {
	return &examRepository{db: db}
}

const examColumns = `id, name, type, questions, author, state, owner_id, created_at`

func scanExam(rows pgx.Row, e *model.Exam) error {
	return rows.Scan(&e.ID, &e.Name, &e.Type, &e.Questions, &e.Author, &e.State, &e.OwnerID, &e.CreatedAt)
}

// List returns the global/admin bank only (owner_id IS NULL).
func (r *examRepository) List(ctx context.Context, limit, offset int) ([]model.Exam, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM exams WHERE owner_id IS NULL`).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx, `SELECT `+examColumns+` FROM exams WHERE owner_id IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	exams := []model.Exam{}
	for rows.Next() {
		var e model.Exam
		if err := scanExam(rows, &e); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		exams = append(exams, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return exams, total, nil
}

func (r *examRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.Exam, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM exams WHERE owner_id = $1`, ownerID).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx, `SELECT `+examColumns+` FROM exams WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, ownerID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	exams := []model.Exam{}
	for rows.Next() {
		var e model.Exam
		if err := scanExam(rows, &e); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		exams = append(exams, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return exams, total, nil
}

func (r *examRepository) Get(ctx context.Context, id uuid.UUID) (*model.Exam, error) {
	var e model.Exam
	if err := scanExam(r.db.QueryRow(ctx, `SELECT `+examColumns+` FROM exams WHERE id = $1`, id), &e); err != nil {
		return nil, apperror.NotFound("exam not found")
	}
	return &e, nil
}

func (r *examRepository) Create(ctx context.Context, e *model.Exam) error {
	const query = `
		INSERT INTO exams (name, type, questions, author, state, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query, e.Name, e.Type, e.Questions, e.Author, e.State, e.OwnerID).
		Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *examRepository) Update(ctx context.Context, e *model.Exam) (*model.Exam, error) {
	const query = `
		UPDATE exams SET name = $2, type = $3, questions = $4, state = $5
		WHERE id = $1
		RETURNING ` + examColumns

	var out model.Exam
	if err := scanExam(r.db.QueryRow(ctx, query, e.ID, e.Name, e.Type, e.Questions, e.State), &out); err != nil {
		return nil, apperror.NotFound("exam not found")
	}
	return &out, nil
}

func (r *examRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM exams WHERE id = $1`, id)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("exam not found")
	}
	return nil
}
