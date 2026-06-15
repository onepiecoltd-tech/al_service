package handler

import (
	"fmt"
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type ProfileHandler struct {
	profiles service.ProfileService
}

func NewProfileHandler(profiles service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

type profileResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Handle string `json:"handle"`
	Plan   string `json:"plan"`
	Coins  int    `json:"coins"`
	Elo    int    `json:"elo"`
	Rank   string `json:"rank"`
	Streak int    `json:"streak"`
	Role   string `json:"role"`
	Joined string `json:"joined"`
}

type profileEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data profileResponse `json:"data"`
}

// Me godoc
//
//	@Summary		Get the authenticated user's profile
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	profileEnvelope
//	@Failure		401	{object}	errorEnvelope	"missing or invalid token"
//	@Router			/api/v1/me [get]
func (h *ProfileHandler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	u, err := h.profiles.Get(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, profileResponse{
		ID:     u.ID.String(),
		Email:  u.Email,
		Name:   u.DisplayName,
		Handle: u.Handle,
		Plan:   u.Plan,
		Coins:  u.Coins,
		Elo:    u.Elo,
		Rank:   rankFromElo(u.Elo),
		Streak: u.Streak,
		Role:   u.Role,
		Joined: fmt.Sprintf("Tháng %d, %d", int(u.CreatedAt.Month()), u.CreatedAt.Year()),
	})
}

func rankFromElo(elo int) string {
	switch {
	case elo >= 1500:
		return "Bạch kim"
	case elo >= 1400:
		return "Vàng"
	case elo >= 1200:
		return "Bạc"
	default:
		return "Đồng"
	}
}
