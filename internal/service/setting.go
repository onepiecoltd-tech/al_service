package service

import (
	"context"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type SettingService interface {
	List(ctx context.Context) ([]model.Setting, error)
	Update(ctx context.Context, key string, value bool) (*model.Setting, error)
}

type settingService struct {
	repo repository.SettingRepository
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) List(ctx context.Context) ([]model.Setting, error) {
	return s.repo.List(ctx)
}

func (s *settingService) Update(ctx context.Context, key string, value bool) (*model.Setting, error) {
	return s.repo.Update(ctx, key, value)
}
