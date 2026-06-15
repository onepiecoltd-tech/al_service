package service

import (
	"context"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const leaderboardSize = 20

type LeaderboardService interface {
	Top(ctx context.Context) ([]model.User, error)
}

type leaderboardService struct {
	users repository.UserRepository
}

func NewLeaderboardService(users repository.UserRepository) LeaderboardService {
	return &leaderboardService{users: users}
}

func (s *leaderboardService) Top(ctx context.Context) ([]model.User, error) {
	return s.users.TopByElo(ctx, leaderboardSize)
}
