package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AuthHandler struct {
	auth service.AuthService
}

func NewAuthHandler(auth service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email" example:"minhanh@email.com"`
	Password string `json:"password" example:"password"`
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
