package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type ProfileService interface {
	Get(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetPrefs(ctx context.Context, id uuid.UUID) (map[string]bool, error)
	SetPrefs(ctx context.Context, id uuid.UUID, prefs map[string]bool) error
}

type profileService struct {
	users repository.UserRepository
}

func NewProfileService(users repository.UserRepository) ProfileService {
	return &profileService{users: users}
}

func (s *profileService) Get(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *profileService) GetPrefs(ctx context.Context, id uuid.UUID) (map[string]bool, error) {
	return s.users.GetPrefs(ctx, id)
}

func (s *profileService) SetPrefs(ctx context.Context, id uuid.UUID, prefs map[string]bool) error {
	return s.users.SetPrefs(ctx, id, prefs)
}
