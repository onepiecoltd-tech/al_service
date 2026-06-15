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

type AdminReportHandler struct {
	reports service.ReportService
}

func NewAdminReportHandler(reports service.ReportService) *AdminReportHandler {
	return &AdminReportHandler{reports: reports}
}

type reportListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Report `json:"data"`
}

type resolveRequest struct {
	Action string `json:"action" example:"removed"`
}

// List godoc
//
//	@Summary	List moderation reports (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		status	query		string	false	"open (default) or resolved"
//	@Success	200		{object}	reportListEnvelope
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/admin/reports [get]
func (h *AdminReportHandler) List(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reports.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, reports)
}

// Resolve godoc
//
//	@Summary	Resolve a report — dismiss/hide/remove (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string			true	"Report ID"
//	@Param		body	body		resolveRequest	true	"Action"
//	@Success	200		{object}	reportListEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/admin/reports/{id}/resolve [post]
func (h *AdminReportHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid report id"))
		return
	}
	var req resolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	report, err := h.reports.Resolve(r.Context(), id, req.Action)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, report)
}
