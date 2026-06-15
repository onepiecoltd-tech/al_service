package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type FriendService interface {
	List(ctx context.Context, userID uuid.UUID) ([]model.User, error)
}

type friendService struct {
	users repository.UserRepository
}

func NewFriendService(users repository.UserRepository) FriendService {
	return &friendService{users: users}
}

func (s *friendService) List(ctx context.Context, userID uuid.UUID) ([]model.User, error) {
	return s.users.ListFriends(ctx, userID)
}
