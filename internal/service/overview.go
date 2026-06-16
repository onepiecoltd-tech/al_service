package service

import (
	"context"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type OverviewService interface {
	Get(ctx context.Context) (*model.Overview, error)
}

type overviewService struct {
	repo repository.OverviewRepository
}

func NewOverviewService(repo repository.OverviewRepository) OverviewService {
	return &overviewService{repo: repo}
}

func (s *overviewService) Get(ctx context.Context) (*model.Overview, error) {
	return s.repo.Get(ctx)
}
