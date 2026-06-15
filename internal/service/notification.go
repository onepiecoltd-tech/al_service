package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type NotificationService interface {
	List(ctx context.Context, userID uuid.UUID) ([]model.Notification, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type notificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) List(ctx context.Context, userID uuid.UUID) ([]model.Notification, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}
