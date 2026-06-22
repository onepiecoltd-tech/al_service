package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const tokenTTL = 24 * time.Hour

type AuthService interface {
	Login(ctx context.Context, email, password string) (string, *model.User, error)
	Register(ctx context.Context, email, name, password string) (string, *model.User, error)
	// LoginWithGoogle verifies a Google Identity Services ID token, finds or
	// creates the matching account, and returns a signed JWT plus the user.
	LoginWithGoogle(ctx context.Context, idToken string) (string, *model.User, error)
}

type authService struct {
	users          repository.UserRepository
	jwtSecret      []byte
	googleClientID string
}

func NewAuthService(users repository.UserRepository, jwtSecret, googleClientID string) AuthService {
	return &authService{users: users, jwtSecret: []byte(jwtSecret), googleClientID: googleClientID}
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
	if user.Status != "active" {
		return "", nil, apperror.Forbidden("tài khoản đã bị khóa")
	}

	token, err := s.issueToken(user)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}
	return token, user, nil
}

// Register creates a new account and returns a signed JWT (auto-login).
func (s *authService) Register(ctx context.Context, email, name, password string) (string, *model.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if email == "" || name == "" {
		return "", nil, apperror.BadRequest("email and name are required")
	}
	if len(password) < 6 {
		return "", nil, apperror.BadRequest("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}

	user, err := s.createUser(ctx, email, name, string(hash))
	if err != nil {
		return "", nil, err
	}

	token, err := s.issueToken(user)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}
	return token, user, nil
}

// LoginWithGoogle verifies the Google ID token (signature, issuer, audience,
// expiry are all checked by idtoken.Validate), then finds or creates the
// matching account and returns a signed JWT.
func (s *authService) LoginWithGoogle(ctx context.Context, idTokenStr string) (string, *model.User, error) {
	if s.googleClientID == "" {
		return "", nil, apperror.Internal(fmt.Errorf("thiếu GOOGLE_CLIENT_ID trên server"))
	}
	payload, err := idtoken.Validate(ctx, idTokenStr, s.googleClientID)
	if err != nil {
		return "", nil, apperror.Unauthorized("invalid Google token")
	}
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	email, _ := payload.Claims["email"].(string)
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !emailVerified {
		return "", nil, apperror.Unauthorized("Google email not verified")
	}
	name, _ := payload.Claims["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = email
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if apperror.StatusCode(err) != http.StatusNotFound {
			return "", nil, err
		}
		// New account: random password the user will never know, since they
		// only ever sign in via Google.
		randomHash, hErr := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
		if hErr != nil {
			return "", nil, apperror.Internal(hErr)
		}
		user, err = s.createUser(ctx, email, name, string(randomHash))
		if err != nil {
			return "", nil, err
		}
	}
	if user.Status != "active" {
		return "", nil, apperror.Forbidden("tài khoản đã bị khóa")
	}

	token, err := s.issueToken(user)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}
	return token, user, nil
}

func (s *authService) createUser(ctx context.Context, email, name, passwordHash string) (*model.User, error) {
	handle := "@" + email
	if i := strings.IndexByte(email, '@'); i > 0 {
		handle = "@" + email[:i]
	}
	user := &model.User{
		Email:        email,
		DisplayName:  name,
		PasswordHash: passwordHash,
		Handle:       handle,
		Plan:         "Free",
		Role:         "user",
		Status:       "active",
	}
	if err := s.users.Insert(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) issueToken(user *model.User) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
