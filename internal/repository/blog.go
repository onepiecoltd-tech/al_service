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

type BlogRepository interface {
	List(ctx context.Context) ([]model.BlogPost, error)
	Get(ctx context.Context, id uuid.UUID) (*model.BlogPost, error)
	Create(ctx context.Context, p *model.BlogPost) error
	Update(ctx context.Context, p *model.BlogPost) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type blogRepository struct {
	db *pgxpool.Pool
}

func NewBlogRepository(db *pgxpool.Pool) BlogRepository {
	return &blogRepository{db: db}
}

const blogColumns = `id, title, category, author, excerpt, body, reads, comments, status, created_at`

func scanBlog(row pgx.Row) (*model.BlogPost, error) {
	var p model.BlogPost
	err := row.Scan(
		&p.ID, &p.Title, &p.Category, &p.Author, &p.Excerpt,
		&p.Body, &p.Reads, &p.Comments, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("blog post not found")
		}
		return nil, apperror.Internal(err)
	}
	return &p, nil
}

func (r *blogRepository) List(ctx context.Context) ([]model.BlogPost, error) {
	rows, err := r.db.Query(ctx, `SELECT `+blogColumns+` FROM blog_posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	posts := []model.BlogPost{}
	for rows.Next() {
		var p model.BlogPost
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Category, &p.Author, &p.Excerpt,
			&p.Body, &p.Reads, &p.Comments, &p.Status, &p.CreatedAt,
		); err != nil {
			return nil, apperror.Internal(err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return posts, nil
}

func (r *blogRepository) Get(ctx context.Context, id uuid.UUID) (*model.BlogPost, error) {
	return scanBlog(r.db.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts WHERE id = $1`, id))
}

func (r *blogRepository) Create(ctx context.Context, p *model.BlogPost) error {
	const query = `
		INSERT INTO blog_posts (title, category, author, excerpt, body, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, reads, comments, created_at`
	err := r.db.QueryRow(ctx, query, p.Title, p.Category, p.Author, p.Excerpt, p.Body, p.Status).
		Scan(&p.ID, &p.Reads, &p.Comments, &p.CreatedAt)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *blogRepository) Update(ctx context.Context, p *model.BlogPost) error {
	const query = `
		UPDATE blog_posts
		SET title = $2, category = $3, excerpt = $4, body = $5, status = $6
		WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, p.ID, p.Title, p.Category, p.Excerpt, p.Body, p.Status)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("blog post not found")
	}
	return nil
}

func (r *blogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM blog_posts WHERE id = $1`, id)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("blog post not found")
	}
	return nil
}
