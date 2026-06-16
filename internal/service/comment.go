package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type CommentService interface {
	List(ctx context.Context, postID uuid.UUID) ([]model.Comment, error)
	Create(ctx context.Context, postID uuid.UUID, author, body string) (*model.Comment, error)
}

type commentService struct {
	repo repository.CommentRepository
}

func NewCommentService(repo repository.CommentRepository) CommentService {
	return &commentService{repo: repo}
}

func (s *commentService) List(ctx context.Context, postID uuid.UUID) ([]model.Comment, error) {
	return s.repo.ListByPost(ctx, postID)
}

func (s *commentService) Create(ctx context.Context, postID uuid.UUID, author, body string) (*model.Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, apperror.BadRequest("comment cannot be empty")
	}
	c := &model.Comment{PostID: postID, Author: author, Body: body}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
