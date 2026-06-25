package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

// AdminBackfillHandler lets an admin manually kick off the AI answer-backfill
// job instead of waiting for its hourly tick.
type AdminBackfillHandler struct {
	job *service.AnswerBackfiller
}

func NewAdminBackfillHandler(job *service.AnswerBackfiller) *AdminBackfillHandler {
	return &AdminBackfillHandler{job: job}
}

// TriggerAnswers godoc
//
//	@Summary	Manually trigger the AI answer-backfill job (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/v1/admin/questions/backfill-answers [post]
func (h *AdminBackfillHandler) TriggerAnswers(w http.ResponseWriter, r *http.Request) {
	started := h.job.Trigger()
	httputil.OK(w, map[string]any{"started": started})
}
