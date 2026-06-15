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

type AdminExamHandler struct {
	exams    service.ExamService
	profiles service.ProfileService
}

func NewAdminExamHandler(exams service.ExamService, profiles service.ProfileService) *AdminExamHandler {
	return &AdminExamHandler{exams: exams, profiles: profiles}
}

type examRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Questions int    `json:"questions"`
	State     string `json:"state" example:"draft"`
}

type examListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Exam `json:"data"`
}

type examEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data model.Exam `json:"data"`
}

// List godoc
//
//	@Summary	List exams (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		page	query		int	false	"page (default 1)"
//	@Param		limit	query		int	false	"limit (default 20, max 100)"
//	@Success	200	{object}	examListEnvelope
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/admin/exams [get]
func (h *AdminExamHandler) List(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := httputil.PageParams(r)
	exams, total, err := h.exams.List(r.Context(), limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Paginated(w, exams, page, limit, total)
}

// Create godoc
//
//	@Summary	Create an exam (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		exam	body		examRequest	true	"Exam"
//	@Success	201		{object}	examEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Router		/api/v1/admin/exams [post]
func (h *AdminExamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req examRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	exam := &model.Exam{
		Name:      req.Name,
		Type:      req.Type,
		Questions: req.Questions,
		Author:    h.authorName(r),
		State:     req.State,
	}
	if err := h.exams.Create(r.Context(), exam); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, exam)
}

// Update godoc
//
//	@Summary	Update an exam (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string		true	"Exam ID"
//	@Param		exam	body		examRequest	true	"Exam"
//	@Success	200		{object}	examEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/admin/exams/{id} [put]
func (h *AdminExamHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	var req examRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	exam, err := h.exams.Update(r.Context(), &model.Exam{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		Questions: req.Questions,
		State:     req.State,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, exam)
}

// Delete godoc
//
//	@Summary	Delete an exam (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Exam ID"
//	@Success	204	"deleted"
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/admin/exams/{id} [delete]
func (h *AdminExamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	if err := h.exams.Delete(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminExamHandler) authorName(r *http.Request) string {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return "Admin"
	}
	if u, err := h.profiles.Get(r.Context(), id); err == nil {
		return u.DisplayName
	}
	return "Admin"
}
