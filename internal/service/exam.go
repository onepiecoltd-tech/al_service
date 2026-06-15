package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type ExamService interface {
	List(ctx context.Context, limit, offset int) ([]model.Exam, int, error)
	Create(ctx context.Context, e *model.Exam) error
	Update(ctx context.Context, e *model.Exam) (*model.Exam, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type examService struct {
	repo repository.ExamRepository
}

func NewExamService(repo repository.ExamRepository) ExamService {
	return &examService{repo: repo}
}

func (s *examService) List(ctx context.Context, limit, offset int) ([]model.Exam, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *examService) Create(ctx context.Context, e *model.Exam) error {
	if err := validateExam(e); err != nil {
		return err
	}
	if e.State == "" {
		e.State = "draft"
	}
	return s.repo.Create(ctx, e)
}

func (s *examService) Update(ctx context.Context, e *model.Exam) (*model.Exam, error) {
	if err := validateExam(e); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, e)
}

func (s *examService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateExam(e *model.Exam) error {
	if strings.TrimSpace(e.Name) == "" {
		return apperror.BadRequest("name is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return apperror.BadRequest("type is required")
	}
	return nil
}
