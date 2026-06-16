package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type BadgeHandler struct {
	badges service.BadgeService
}

func NewBadgeHandler(badges service.BadgeService) *BadgeHandler {
	return &BadgeHandler{badges: badges}
}

type badgeListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Badge `json:"data"`
}

// Me godoc
//
//	@Summary	Badges earned by the authenticated user
//	@Tags		profile
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	badgeListEnvelope
//	@Router		/api/v1/me/badges [get]
func (h *BadgeHandler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}
	badges, err := h.badges.ListByUser(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, badges)
}
