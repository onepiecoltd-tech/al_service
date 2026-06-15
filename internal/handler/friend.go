package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type FriendHandler struct {
	friends service.FriendService
}

func NewFriendHandler(friends service.FriendService) *FriendHandler {
	return &FriendHandler{friends: friends}
}

type friendRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Handle   string `json:"handle"`
	Elo      int    `json:"elo"`
	Presence string `json:"presence"`
	Msg      string `json:"msg"`
}

type friendListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []friendRow `json:"data"`
}

// List godoc
//
//	@Summary	List the authenticated user's friends
//	@Tags		friends
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	friendListEnvelope
//	@Failure	401	{object}	errorEnvelope
//	@Router		/api/v1/friends [get]
func (h *FriendHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}

	friends, err := h.friends.List(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	rows := make([]friendRow, len(friends))
	for i, u := range friends {
		rows[i] = friendRow{
			ID:       u.ID.String(),
			Name:     u.DisplayName,
			Handle:   u.Handle,
			Elo:      u.Elo,
			Presence: u.Presence,
			Msg:      u.StatusMsg,
		}
	}
	httputil.OK(w, rows)
}
