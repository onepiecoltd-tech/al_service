package handler

import (
	"encoding/json"
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AdminSettingHandler struct {
	settings service.SettingService
}

func NewAdminSettingHandler(settings service.SettingService) *AdminSettingHandler {
	return &AdminSettingHandler{settings: settings}
}

type settingListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Setting `json:"data"`
}

type updateSettingRequest struct {
	Value bool `json:"value"`
}

// List godoc
//
//	@Summary	List feature settings (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	settingListEnvelope
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/admin/settings [get]
func (h *AdminSettingHandler) List(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, settings)
}

// Update godoc
//
//	@Summary	Toggle a feature setting (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		key		path		string					true	"Setting key"
//	@Param		body	body		updateSettingRequest	true	"Value"
//	@Success	200		{object}	settingListEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/admin/settings/{key} [put]
func (h *AdminSettingHandler) Update(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.Error(w, apperror.BadRequest("missing setting key"))
		return
	}
	var req updateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	s, err := h.settings.Update(r.Context(), key, req.Value)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, s)
}
