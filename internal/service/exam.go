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
	Get(ctx context.Context, id uuid.UUID) (*model.Exam, error)
	Create(ctx context.Context, e *model.Exam) error
	Update(ctx context.Context, e *model.Exam) (*model.Exam, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// Import uses Gemini to extract questions from an uploaded exam file (.pdf or .txt),
	// replaces the exam's question bank, and updates its question count.
	Import(ctx context.Context, examID uuid.UUID, filename string, data []byte) ([]model.Question, error)
	Questions(ctx context.Context, examID uuid.UUID) ([]model.Question, error)
}

type examService struct {
	repo      repository.ExamRepository
	questions repository.QuestionRepository
	ai        *GeminiClient
}

func NewExamService(repo repository.ExamRepository, questions repository.QuestionRepository, ai *GeminiClient) ExamService {
	return &examService{repo: repo, questions: questions, ai: ai}
}

func (s *examService) Import(ctx context.Context, examID uuid.UUID, filename string, data []byte) ([]model.Question, error) {
	exam, err := s.repo.Get(ctx, examID)
	if err != nil {
		return nil, err
	}
	qs, err := s.ai.ExtractQuestions(ctx, filename, data)
	if err != nil {
		return nil, err
	}
	if err := s.questions.ReplaceForExam(ctx, examID, qs); err != nil {
		return nil, err
	}
	exam.Questions = len(qs)
	if _, err := s.repo.Update(ctx, exam); err != nil {
		return nil, err
	}
	return qs, nil
}

func (s *examService) Questions(ctx context.Context, examID uuid.UUID) ([]model.Question, error) {
	return s.questions.ListByExam(ctx, examID)
}

func (s *examService) List(ctx context.Context, limit, offset int) ([]model.Exam, int, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *examService) Get(ctx context.Context, id uuid.UUID) (*model.Exam, error) {
	return s.repo.Get(ctx, id)
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
