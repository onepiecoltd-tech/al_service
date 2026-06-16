package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type WalletHandler struct {
	wallet service.WalletService
}

func NewWalletHandler(wallet service.WalletService) *WalletHandler {
	return &WalletHandler{wallet: wallet}
}

type coinPackListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.CoinPack `json:"data"`
}

type txnListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Transaction `json:"data"`
}

type topupResultEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data service.TopupResult `json:"data"`
}

type topupRequest struct {
	PackID string `json:"pack_id"`
}

type giftRequest struct {
	GiftID string `json:"gift_id"`
}

// CoinPacks godoc
//
//	@Summary	List coin packs
//	@Tags		wallet
//	@Produce	json
//	@Success	200	{object}	coinPackListEnvelope
//	@Router		/api/v1/coin-packs [get]
func (h *WalletHandler) CoinPacks(w http.ResponseWriter, r *http.Request) {
	packs, err := h.wallet.CoinPacks(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, packs)
}

// Transactions godoc
//
//	@Summary	My wallet transactions
//	@Tags		wallet
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	txnListEnvelope
//	@Router		/api/v1/wallet/transactions [get]
func (h *WalletHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	txns, err := h.wallet.Transactions(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, txns)
}

// Topup godoc
//
//	@Summary	Buy a coin pack (mock PayOS)
//	@Tags		wallet
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		topupRequest	true	"Pack id"
//	@Success	200		{object}	topupResultEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/wallet/topup [post]
func (h *WalletHandler) Topup(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	var req topupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	packID, err := uuid.Parse(req.PackID)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid pack id"))
		return
	}
	res, err := h.wallet.Topup(r.Context(), id, packID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, res)
}

// Gift godoc
//
//	@Summary	Send a gift (spend coins)
//	@Tags		wallet
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		giftRequest	true	"Gift id"
//	@Success	200		{object}	topupResultEnvelope
//	@Failure	400		{object}	errorEnvelope	"insufficient coins"
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/wallet/gift [post]
func (h *WalletHandler) Gift(w http.ResponseWriter, r *http.Request) {
	id, _ := middleware.UserIDFromContext(r.Context())
	var req giftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	giftID, err := uuid.Parse(req.GiftID)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid gift id"))
		return
	}
	res, err := h.wallet.Gift(r.Context(), id, giftID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, res)
}
