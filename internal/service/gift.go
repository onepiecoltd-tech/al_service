package service

import (
	"context"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type GiftService interface {
	List(ctx context.Context) ([]model.Gift, error)
}

type giftService struct {
	repo repository.GiftRepository
}

func NewGiftService(repo repository.GiftRepository) GiftService {
	return &giftService{repo: repo}
}

func (s *giftService) List(ctx context.Context) ([]model.Gift, error) {
	return s.repo.List(ctx)
}
