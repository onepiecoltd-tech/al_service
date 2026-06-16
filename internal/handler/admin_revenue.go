package handler

import (
	"net/http"

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
