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

type DuelHandler struct {
	duels service.DuelService
}

func NewDuelHandler(duels service.DuelService) *DuelHandler {
	return &DuelHandler{duels: duels}
}

// List godoc
//
//	@Summary	List the authenticated user's recent duels
//	@Tags		duels
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/v1/duels [get]
func (h *DuelHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	duels, err := h.duels.List(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, duels)
}

type challengeRequest struct {
	OpponentID string `json:"opponent_id"`
	Prompt     string `json:"prompt"`
	Score      int    `json:"score"`
}

// Challenge godoc
//
//	@Summary	Challenge a friend to a pronunciation duel
//	@Tags		duels
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		challengeRequest	true	"Challenge"
//	@Success	201		{object}	map[string]interface{}
//	@Failure	400		{object}	errorEnvelope
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/duels [post]
func (h *DuelHandler) Challenge(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	var req challengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	opponentID, err := uuid.Parse(req.OpponentID)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid opponent id"))
		return
	}
	d, err := h.duels.Challenge(r.Context(), id, opponentID, req.Prompt, req.Score)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, d)
}

type respondRequest struct {
	Score int `json:"score"`
}

// Respond godoc
//
//	@Summary	Respond to a duel by recording your score (resolves the match)
//	@Tags		duels
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string			true	"Duel id"
//	@Param		body	body		respondRequest	true	"Your score"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/duels/{id}/respond [post]
func (h *DuelHandler) Respond(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	duelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid duel id"))
		return
	}
	var req respondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	d, err := h.duels.Respond(r.Context(), id, duelID, req.Score)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, d)
}

// Decline godoc
//
//	@Summary	Decline a pending duel
//	@Tags		duels
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Duel id"
//	@Success	204	"declined"
//	@Router		/api/v1/duels/{id}/decline [post]
func (h *DuelHandler) Decline(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	duelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid duel id"))
		return
	}
	if err := h.duels.Decline(r.Context(), id, duelID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
