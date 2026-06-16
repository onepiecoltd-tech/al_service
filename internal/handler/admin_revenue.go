package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AdminRevenueHandler struct {
	wallet service.WalletService
}

func NewAdminRevenueHandler(wallet service.WalletService) *AdminRevenueHandler {
	return &AdminRevenueHandler{wallet: wallet}
}

type adminTxnListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.AdminTransaction `json:"data"`
}

type revenueEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data model.RevenueSummary `json:"data"`
}

// Transactions godoc
//
//	@Summary	All transactions (admin, paginated)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		page	query		int	false	"page"
//	@Param		limit	query		int	false	"limit"
//	@Success	200		{object}	adminTxnListEnvelope
//	@Router		/api/v1/admin/transactions [get]
func (h *AdminRevenueHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := httputil.PageParams(r)
	txns, total, err := h.wallet.AllTransactions(r.Context(), limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Paginated(w, txns, page, limit, total)
}

type coinPackRequest struct {
	VND     int  `json:"vnd"`
	Coins   int  `json:"coins"`
	Popular bool `json:"popular"`
}

// CreatePack godoc
//
//	@Summary	Create a coin pack (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		coinPackRequest	true	"Pack"
//	@Success	201		{object}	map[string]model.CoinPack
//	@Router		/api/v1/admin/coin-packs [post]
func (h *AdminRevenueHandler) CreatePack(w http.ResponseWriter, r *http.Request) {
	var req coinPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	p, err := h.wallet.CreateCoinPack(r.Context(), req.VND, req.Coins, req.Popular)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, p)
}

// UpdatePack godoc
//
//	@Summary	Update a coin pack (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string			true	"Pack ID"
//	@Param		body	body		coinPackRequest	true	"Pack"
//	@Success	200		{object}	map[string]model.CoinPack
//	@Router		/api/v1/admin/coin-packs/{id} [put]
func (h *AdminRevenueHandler) UpdatePack(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid pack id"))
		return
	}
	var req coinPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	p, err := h.wallet.UpdateCoinPack(r.Context(), id, req.VND, req.Coins, req.Popular)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, p)
}

// DeletePack godoc
//
//	@Summary	Delete a coin pack (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Pack ID"
//	@Success	204	"deleted"
//	@Router		/api/v1/admin/coin-packs/{id} [delete]
func (h *AdminRevenueHandler) DeletePack(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid pack id"))
		return
	}
	if err := h.wallet.DeleteCoinPack(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Revenue godoc
//
//	@Summary	Revenue summary (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	revenueEnvelope
//	@Router		/api/v1/admin/revenue [get]
func (h *AdminRevenueHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	s, err := h.wallet.Revenue(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, s)
}
