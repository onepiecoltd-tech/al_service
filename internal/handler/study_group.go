package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type StudyGroupHandler struct {
	groups service.StudyGroupService
}

func NewStudyGroupHandler(groups service.StudyGroupService) *StudyGroupHandler {
	return &StudyGroupHandler{groups: groups}
}

// List godoc
//
//	@Summary	List the authenticated user's study groups
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/v1/groups [get]
func (h *StudyGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	groups, err := h.groups.List(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, groups)
}

type createGroupRequest struct {
	Name string `json:"name"`
}

// Create godoc
//
//	@Summary	Create a study group (creator joins automatically)
//	@Tags		groups
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		createGroupRequest	true	"Group name"
//	@Success	201		{object}	map[string]interface{}
//	@Failure	400		{object}	errorEnvelope
//	@Router		/api/v1/groups [post]
func (h *StudyGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	g, err := h.groups.Create(r.Context(), id, req.Name)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, g)
}

type joinGroupRequest struct {
	Code string `json:"code"`
}

// Join godoc
//
//	@Summary	Join a study group by code
//	@Tags		groups
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		joinGroupRequest	true	"Join code"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/groups/join [post]
func (h *StudyGroupHandler) Join(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var req joinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	g, pending, err := h.groups.Join(r.Context(), id, req.Code)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"group": g, "pending": pending})
}

type groupMember struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Handle   string `json:"handle"`
	Elo      int    `json:"elo"`
	Presence string `json:"presence"`
}

// Members godoc
//
//	@Summary	List a study group's members (members only)
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Group id"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/groups/{id}/members [get]
func (h *StudyGroupHandler) Members(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid group id"))
		return
	}
	users, err := h.groups.Members(r.Context(), id, groupID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows := make([]groupMember, len(users))
	for i, u := range users {
		rows[i] = groupMember{ID: u.ID.String(), Name: u.DisplayName, Handle: u.Handle, Elo: u.Elo, Presence: u.Presence}
	}
	httputil.OK(w, rows)
}

// PendingRequests godoc
//
//	@Summary	List pending join requests for a group (owner only)
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Group id"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/groups/{id}/requests [get]
func (h *StudyGroupHandler) PendingRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid group id"))
		return
	}
	users, err := h.groups.PendingRequests(r.Context(), id, groupID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows := make([]groupMember, len(users))
	for i, u := range users {
		rows[i] = groupMember{ID: u.ID.String(), Name: u.DisplayName, Handle: u.Handle, Elo: u.Elo, Presence: u.Presence}
	}
	httputil.OK(w, rows)
}

// Approve godoc
//
//	@Summary	Approve a pending join request (owner only)
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path	string	true	"Group id"
//	@Param		userId	path	string	true	"Requesting user id"
//	@Success	204		"approved"
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/groups/{id}/requests/{userId}/approve [post]
func (h *StudyGroupHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.actOnRequest(w, r, h.groups.Approve)
}

// Reject godoc
//
//	@Summary	Reject a pending join request (owner only)
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path	string	true	"Group id"
//	@Param		userId	path	string	true	"Requesting user id"
//	@Success	204		"rejected"
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/groups/{id}/requests/{userId}/reject [post]
func (h *StudyGroupHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.actOnRequest(w, r, h.groups.Reject)
}

func (h *StudyGroupHandler) actOnRequest(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, ownerID, groupID, userID uuid.UUID) error) {
	ownerID, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid group id"))
		return
	}
	userID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid user id"))
		return
	}
	if err := action(r.Context(), ownerID, groupID, userID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Leave godoc
//
//	@Summary	Leave a study group
//	@Tags		groups
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Group id"
//	@Success	204	"left"
//	@Router		/api/v1/groups/{id}/leave [delete]
func (h *StudyGroupHandler) Leave(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid group id"))
		return
	}
	if err := h.groups.Leave(r.Context(), id, groupID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
