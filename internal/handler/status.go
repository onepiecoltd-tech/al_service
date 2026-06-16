package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type StatusHandler struct {
	settings service.SettingService
}

func NewStatusHandler(settings service.SettingService) *StatusHandler {
	return &StatusHandler{settings: settings}
}

type statusResponse struct {
	Maintenance bool `json:"maintenance"`
	AllowSignup bool `json:"allow_signup"`
}

type statusEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data statusResponse `json:"data"`
}

// Status godoc
//
//	@Summary		Public app status (feature flags that affect all clients)
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	statusEnvelope
//	@Router			/api/v1/status [get]
func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	list, err := h.settings.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	flags := make(map[string]bool, len(list))
	for _, s := range list {
		flags[s.Key] = s.Value
	}
	httputil.OK(w, statusResponse{
		Maintenance: flags["maintenance"],
		AllowSignup: flags["allow_signup"],
	})
}
