package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AdminOverviewHandler struct {
	overview service.OverviewService
}

func NewAdminOverviewHandler(overview service.OverviewService) *AdminOverviewHandler {
	return &AdminOverviewHandler{overview: overview}
}

type overviewEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data model.Overview `json:"data"`
}

// Get godoc
//
//	@Summary	Admin dashboard stats (counts + recent signups)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	overviewEnvelope
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/admin/overview [get]
func (h *AdminOverviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	o, err := h.overview.Get(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, o)
}
