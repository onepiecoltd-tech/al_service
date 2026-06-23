package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type BlogService interface {
	List(ctx context.Context, category, status string, limit, offset int) ([]model.BlogPost, int, error)
	Get(ctx context.Context, id uuid.UUID) (*model.BlogPost, error)
	Create(ctx context.Context, p *model.BlogPost) error
	Update(ctx context.Context, p *model.BlogPost) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type blogService struct {
	repo repository.BlogRepository
}

func NewBlogService(repo repository.BlogRepository) BlogService {
	return &blogService{repo: repo}
}

func (s *blogService) List(ctx context.Context, category, status string, limit, offset int) ([]model.BlogPost, int, error) {
	return s.repo.List(ctx, category, status, limit, offset)
}

func (s *blogService) Get(ctx context.Context, id uuid.UUID) (*model.BlogPost, error) {
	return s.repo.Get(ctx, id)
}

func (s *blogService) Create(ctx context.Context, p *model.BlogPost) error {
	if err := validate(p); err != nil {
		return err
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	return s.repo.Create(ctx, p)
}

func (s *blogService) Update(ctx context.Context, p *model.BlogPost) error {
	if err := validate(p); err != nil {
		return err
	}
	return s.repo.Update(ctx, p)
}

func (s *blogService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validate(p *model.BlogPost) error {
	if strings.TrimSpace(p.Title) == "" {
		return apperror.BadRequest("title is required")
	}
	if strings.TrimSpace(p.Category) == "" {
		return apperror.BadRequest("category is required")
	}
	return nil
}
