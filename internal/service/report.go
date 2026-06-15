package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

var validActions = map[string]bool{"dismissed": true, "hidden": true, "removed": true}

type ReportService interface {
	List(ctx context.Context, status string) ([]model.Report, error)
	Resolve(ctx context.Context, id uuid.UUID, action string) (*model.Report, error)
}

type reportService struct {
	repo repository.ReportRepository
}

func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) List(ctx context.Context, status string) ([]model.Report, error) {
	if status != "resolved" {
		status = "open"
	}
	return s.repo.ListByStatus(ctx, status)
}

func (s *reportService) Resolve(ctx context.Context, id uuid.UUID, action string) (*model.Report, error) {
	if !validActions[action] {
		return nil, apperror.BadRequest("invalid action")
	}
	return s.repo.Resolve(ctx, id, action)
}
