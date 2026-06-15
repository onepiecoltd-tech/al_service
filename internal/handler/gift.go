package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type GiftHandler struct {
	gifts service.GiftService
}

func NewGiftHandler(gifts service.GiftService) *GiftHandler {
	return &GiftHandler{gifts: gifts}
}

type giftListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Gift `json:"data"`
}

// List godoc
//
//	@Summary	Gift catalog
//	@Tags		gifts
//	@Produce	json
//	@Success	200	{object}	giftListEnvelope
//	@Router		/api/v1/gifts [get]
func (h *GiftHandler) List(w http.ResponseWriter, r *http.Request) {
	gifts, err := h.gifts.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, gifts)
}
