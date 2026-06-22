package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AuthHandler struct {
	auth     service.AuthService
	settings service.SettingService
}

func NewAuthHandler(auth service.AuthService, settings service.SettingService) *AuthHandler {
	return &AuthHandler{auth: auth, settings: settings}
}

type loginRequest struct {
	Email    string `json:"email" example:"minhanh@email.com"`
	Password string `json:"password" example:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

type loginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// loginEnvelope documents the success body: httputil.OK wraps the payload in
// a "data" key. Used for Swagger only.
type loginEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data loginResponse `json:"data"`
}

// errorEnvelope documents the failure body produced by httputil.Error.
type errorEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Error string `json:"error" example:"invalid email or password"`
}

// Login godoc
//
//	@Summary		Log in with email and password
//	@Description	Authenticates a user and returns a JWT valid for 24 hours.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		loginRequest	true	"Login credentials"
//	@Success		200			{object}	loginEnvelope	"JWT and the authenticated user"
//	@Failure		400			{object}	errorEnvelope	"malformed request"
//	@Failure		401			{object}	errorEnvelope	"invalid email or password"
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		httputil.Error(w, apperror.BadRequest("email and password are required"))
		return
	}

	token, user, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, loginResponse{Token: token, User: user})
}

// Register godoc
//
//	@Summary		Register a new account
//	@Description	Creates an account (if signups are open) and returns a JWT.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Registration"
//	@Success		201		{object}	loginEnvelope
//	@Failure		400		{object}	errorEnvelope
//	@Failure		403		{object}	errorEnvelope	"signups disabled"
//	@Failure		409		{object}	errorEnvelope	"email exists"
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.signupAllowed(r.Context()) {
		httputil.Error(w, apperror.Forbidden("đăng ký hiện đang tạm đóng"))
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	token, user, err := h.auth.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, loginResponse{Token: token, User: user})
}

// GoogleLogin godoc
//
//	@Summary		Log in with a Google ID token
//	@Description	Verifies a Google Identity Services ID token and returns a JWT, creating the account on first sign-in.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		googleLoginRequest	true	"Google ID token"
//	@Success		200		{object}	loginEnvelope
//	@Failure		400		{object}	errorEnvelope
//	@Failure		401		{object}	errorEnvelope	"invalid Google token"
//	@Router			/api/v1/auth/google [post]
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.IDToken) == "" {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	token, user, err := h.auth.LoginWithGoogle(r.Context(), req.IDToken)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, loginResponse{Token: token, User: user})
}

func (h *AuthHandler) signupAllowed(ctx context.Context) bool {
	list, err := h.settings.List(ctx)
	if err != nil {
		return false
	}
	for _, s := range list {
		if s.Key == "allow_signup" {
			return s.Value
		}
	}
	return true
}
