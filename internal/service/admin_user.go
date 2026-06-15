package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

type NewUserInput struct {
	Email       string
	DisplayName string
	Password    string
	Handle      string
	Plan        string
	Role        string
}

type AdminUserService interface {
	List(ctx context.Context, q, plan string, limit, offset int) ([]model.User, int, error)
	IsAdmin(ctx context.Context, id uuid.UUID) (bool, error)
	Create(ctx context.Context, in NewUserInput) (*model.User, error)
	Update(ctx context.Context, id uuid.UUID, plan, role, status string) (*model.User, error)
}

type adminUserService struct {
	users repository.UserRepository
}

func NewAdminUserService(users repository.UserRepository) AdminUserService {
	return &adminUserService{users: users}
}

func (s *adminUserService) List(ctx context.Context, q, plan string, limit, offset int) ([]model.User, int, error) {
	plan = strings.ToLower(plan)
	if plan == "all" {
		plan = ""
	}
	return s.users.ListAll(ctx, strings.TrimSpace(q), plan, limit, offset)
}

func (s *adminUserService) IsAdmin(ctx context.Context, id uuid.UUID) (bool, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	return u.Role == "admin" || u.Role == "mod", nil
}

func (s *adminUserService) Create(ctx context.Context, in NewUserInput) (*model.User, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || strings.TrimSpace(in.DisplayName) == "" {
		return nil, apperror.BadRequest("email and name are required")
	}
	if len(in.Password) < 6 {
		return nil, apperror.BadRequest("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	u := &model.User{
		Email:        in.Email,
		DisplayName:  in.DisplayName,
		PasswordHash: string(hash),
		Handle:       in.Handle,
		Plan:         orDefault(in.Plan, "Free"),
		Role:         orDefault(in.Role, "user"),
		Status:       "active",
	}
	if err := s.users.Insert(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *adminUserService) Update(ctx context.Context, id uuid.UUID, plan, role, status string) (*model.User, error) {
	return s.users.UpdateAdminFields(ctx, id, orDefault(plan, "Free"), orDefault(role, "user"), orDefault(status, "active"))
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
