package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type CommentRepository interface {
	ListByPost(ctx context.Context, postID uuid.UUID) ([]model.Comment, error)
	Create(ctx context.Context, c *model.Comment) error
}

type commentRepository struct {
	db *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) ListByPost(ctx context.Context, postID uuid.UUID) ([]model.Comment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, post_id, author, body, created_at FROM comments WHERE post_id = $1 ORDER BY created_at`, postID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	comments := []model.Comment{}
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return comments, nil
}

// Create inserts the comment and bumps the post's denormalized comment count
// in one transaction. Fails if the post does not exist (FK violation).
func (r *commentRepository) Create(ctx context.Context, c *model.Comment) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blog_posts WHERE id = $1)`, c.PostID).Scan(&exists); err != nil {
		return apperror.Internal(err)
	}
	if !exists {
		return apperror.NotFound("blog post not found")
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO comments (post_id, author, body) VALUES ($1, $2, $3) RETURNING id, created_at`,
		c.PostID, c.Author, c.Body).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return apperror.Internal(err)
	}

	if _, err := tx.Exec(ctx, `UPDATE blog_posts SET comments = comments + 1 WHERE id = $1`, c.PostID); err != nil {
		return apperror.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal(err)
	}
	return nil
}
