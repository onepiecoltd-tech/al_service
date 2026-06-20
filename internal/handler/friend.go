package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

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
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
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

type userMini struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Handle string `json:"handle"`
	Elo    int    `json:"elo"`
}

type addFriendRequest struct {
	FriendID string `json:"friend_id"`
}

// Search godoc
//
//	@Summary	Search users to add as friends (excludes self & existing friends)
//	@Tags		friends
//	@Produce	json
//	@Security	BearerAuth
//	@Param		q	query		string	false	"name/email/handle"
//	@Success	200	{object}	map[string][]userMini
//	@Router		/api/v1/users/search [get]
func (h *FriendHandler) Search(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	users, err := h.friends.Search(r.Context(), id, r.URL.Query().Get("q"))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows := make([]userMini, len(users))
	for i, u := range users {
		rows[i] = userMini{ID: u.ID.String(), Name: u.DisplayName, Handle: u.Handle, Elo: u.Elo}
	}
	httputil.OK(w, rows)
}

// Add godoc
//
//	@Summary	Add a friend
//	@Tags		friends
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		addFriendRequest	true	"Friend id"
//	@Success	204		"added"
//	@Failure	400		{object}	errorEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/friends [post]
func (h *FriendHandler) Add(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var req addFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	friendID, err := uuid.Parse(req.FriendID)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid friend id"))
		return
	}
	if err := h.friends.Add(r.Context(), id, friendID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Remove godoc
//
//	@Summary	Remove a friend
//	@Tags		friends
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Friend user id"
//	@Success	204	"removed"
//	@Router		/api/v1/friends/{id} [delete]
func (h *FriendHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	friendID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid friend id"))
		return
	}
	if err := h.friends.Remove(r.Context(), id, friendID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
