package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type ProfileService interface {
	Get(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetPrefs(ctx context.Context, id uuid.UUID) (map[string]bool, error)
	SetPrefs(ctx context.Context, id uuid.UUID, prefs map[string]bool) error
	GetLearningLanguage(ctx context.Context, id uuid.UUID) (string, error)
	// SetLearningLanguage normalizes the code and returns the stored value.
	SetLearningLanguage(ctx context.Context, id uuid.UUID, lang string) (string, error)
	UpdateDisplayName(ctx context.Context, id uuid.UUID, name string) (*model.User, error)
	// Heartbeat marks the user as currently active, for online presence.
	Heartbeat(ctx context.Context, id uuid.UUID) error
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

func (s *profileService) GetLearningLanguage(ctx context.Context, id uuid.UUID) (string, error) {
	return s.users.GetLearningLanguage(ctx, id)
}

func (s *profileService) SetLearningLanguage(ctx context.Context, id uuid.UUID, lang string) (string, error) {
	norm := normalizeLanguage(lang)
	if norm == "" {
		return "", apperror.BadRequest("mã ngôn ngữ không hợp lệ")
	}
	if err := s.users.SetLearningLanguage(ctx, id, norm); err != nil {
		return "", err
	}
	return norm, nil
}

func (s *profileService) UpdateDisplayName(ctx context.Context, id uuid.UUID, name string) (*model.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperror.BadRequest("tên hiển thị không được để trống")
	}
	if len(name) > 60 {
		return nil, apperror.BadRequest("tên hiển thị quá dài (tối đa 60 ký tự)")
	}
	return s.users.UpdateDisplayName(ctx, id, name)
}

func (s *profileService) Heartbeat(ctx context.Context, id uuid.UUID) error {
	return s.users.Touch(ctx, id)
}
