package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const friendSearchLimit = 20

type FriendService interface {
	List(ctx context.Context, userID uuid.UUID) ([]model.User, error)
	Search(ctx context.Context, userID uuid.UUID, q string) ([]model.User, error)
	Add(ctx context.Context, userID, friendID uuid.UUID) error
	Remove(ctx context.Context, userID, friendID uuid.UUID) error
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

func (s *friendService) Search(ctx context.Context, userID uuid.UUID, q string) ([]model.User, error) {
	return s.users.SearchNonFriends(ctx, userID, strings.TrimSpace(q), friendSearchLimit)
}

func (s *friendService) Add(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return apperror.BadRequest("không thể tự kết bạn với chính mình")
	}
	if _, err := s.users.FindByID(ctx, friendID); err != nil {
		return err // 404 if the target doesn't exist
	}
	return s.users.AddFriend(ctx, userID, friendID)
}

func (s *friendService) Remove(ctx context.Context, userID, friendID uuid.UUID) error {
	return s.users.RemoveFriend(ctx, userID, friendID)
}
