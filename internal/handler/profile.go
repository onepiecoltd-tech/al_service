package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/model"
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
	Wins   int    `json:"wins"`
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
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}

	u, err := h.profiles.Get(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, toProfileResponse(u))
}

func toProfileResponse(u *model.User) profileResponse {
	return profileResponse{
		ID:     u.ID.String(),
		Email:  u.Email,
		Name:   u.DisplayName,
		Handle: u.Handle,
		Plan:   u.Plan,
		Coins:  u.Coins,
		Elo:    u.Elo,
		Rank:   rankFromElo(u.Elo),
		Streak: u.Streak,
		Wins:   u.Wins,
		Role:   u.Role,
		Joined: fmt.Sprintf("Tháng %d, %d", int(u.CreatedAt.Month()), u.CreatedAt.Year()),
	}
}

// UpdateMe godoc
//
//	@Summary	Update the authenticated user's display name
//	@Tags		profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		object{name=string}	true	"New display name"
//	@Success	200		{object}	profileEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Router		/api/v1/me [put]
func (h *ProfileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, apperror.BadRequest("dữ liệu không hợp lệ"))
		return
	}
	u, err := h.profiles.UpdateDisplayName(r.Context(), id, body.Name)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, toProfileResponse(u))
}

type prefsEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data map[string]bool `json:"data"`
}

// GetPrefs godoc
//
//	@Summary	Get my privacy preferences
//	@Tags		profile
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	prefsEnvelope
//	@Router		/api/v1/me/prefs [get]
func (h *ProfileHandler) GetPrefs(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	prefs, err := h.profiles.GetPrefs(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, prefs)
}

// SetPrefs godoc
//
//	@Summary	Replace my privacy preferences
//	@Tags		profile
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		map[string]bool	true	"Preferences"
//	@Success	200		{object}	prefsEnvelope
//	@Router		/api/v1/me/prefs [put]
func (h *ProfileHandler) SetPrefs(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var prefs map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	if err := h.profiles.SetPrefs(r.Context(), id, prefs); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, prefs)
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
