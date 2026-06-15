package service

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const tokenTTL = 24 * time.Hour

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, *model.User, error)
}

type authService struct {
	users     repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(users repository.UserRepository, jwtSecret string) AuthService {
	return &authService{users: users, jwtSecret: []byte(jwtSecret)}
}

// Login verifies credentials and returns a signed JWT plus the user.
// Invalid email and invalid password are reported identically to avoid
// leaking which accounts exist.
func (s *authService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if apperror.StatusCode(err) == http.StatusNotFound {
			return "", nil, apperror.Unauthorized("invalid email or password")
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, apperror.Unauthorized("invalid email or password")
	}

	token, err := s.issueToken(user)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}
	return token, user, nil
}

func (s *authService) issueToken(user *model.User) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
