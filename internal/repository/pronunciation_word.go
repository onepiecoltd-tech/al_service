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

type PronunciationWordRepository interface {
	Random(ctx context.Context) (*model.PronunciationWord, error)
	// FindOrCreate looks up a word case-insensitively, inserting it if it
	// doesn't exist yet, so users can practice any word they type.
	FindOrCreate(ctx context.Context, word string) (*model.PronunciationWord, error)
	SetPhonetic(ctx context.Context, id uuid.UUID, phonetic string) error
}

type pronunciationWordRepository struct {
	db *pgxpool.Pool
}

func NewPronunciationWordRepository(db *pgxpool.Pool) PronunciationWordRepository {
	return &pronunciationWordRepository{db: db}
}

func (r *pronunciationWordRepository) FindOrCreate(ctx context.Context, word string) (*model.PronunciationWord, error) {
	var w model.PronunciationWord
	err := r.db.QueryRow(ctx,
		`SELECT id, word, phonetic, created_at FROM pronunciation_words WHERE lower(word) = lower($1) LIMIT 1`, word).
		Scan(&w.ID, &w.Word, &w.Phonetic, &w.CreatedAt)
	if err == nil {
		return &w, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.Internal(err)
	}

	err = r.db.QueryRow(ctx,
		`INSERT INTO pronunciation_words (word) VALUES ($1) RETURNING id, word, phonetic, created_at`, word).
		Scan(&w.ID, &w.Word, &w.Phonetic, &w.CreatedAt)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &w, nil
}

func (r *pronunciationWordRepository) SetPhonetic(ctx context.Context, id uuid.UUID, phonetic string) error {
	if _, err := r.db.Exec(ctx, `UPDATE pronunciation_words SET phonetic = $1 WHERE id = $2`, phonetic, id); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *pronunciationWordRepository) Random(ctx context.Context) (*model.PronunciationWord, error) {
	var w model.PronunciationWord
	err := r.db.QueryRow(ctx,
		`SELECT id, word, phonetic, created_at FROM pronunciation_words ORDER BY random() LIMIT 1`).
		Scan(&w.ID, &w.Word, &w.Phonetic, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("chưa có từ luyện phát âm nào")
		}
		return nil, apperror.Internal(err)
	}
	return &w, nil
}
