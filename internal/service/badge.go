package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type BadgeService interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Badge, error)
}

type badgeService struct {
	repo repository.BadgeRepository
}

func NewBadgeService(repo repository.BadgeRepository) BadgeService {
	return &badgeService{repo: repo}
}

func (s *badgeService) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Badge, error) {
	return s.repo.ListByUser(ctx, userID)
}
