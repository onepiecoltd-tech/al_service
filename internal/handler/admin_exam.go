package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

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

// Get godoc
//
//	@Summary	Get an exam (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Exam ID"
//	@Success	200	{object}	examEnvelope
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/admin/exams/{id} [get]
func (h *AdminExamHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	exam, err := h.exams.Get(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, exam)
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

// Import godoc
//
//	@Summary	Import questions into an exam using AI extraction (admin)
//	@Tags		admin
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string	true	"Exam ID"
//	@Param		file	formData	file	true	"Exam file (.pdf or .txt)"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	errorEnvelope
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/admin/exams/{id}/import [post]
func (h *AdminExamHandler) Import(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	// AI extraction can take 1-2+ minutes for large PDFs — extend past the
	// server's default 30s WriteTimeout for this handler only.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(4 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5 MB cap
	file, header, err := r.FormFile("file")
	if err != nil {
		if err.Error() == "http: request body too large" {
			httputil.Error(w, apperror.BadRequest("tệp vượt quá giới hạn 5MB"))
			return
		}
		httputil.Error(w, apperror.BadRequest("thiếu tệp đề thi (field \"file\")"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("không đọc được tệp tải lên"))
		return
	}
	qs, err := h.exams.Import(r.Context(), id, header.Filename, data)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"imported": len(qs), "questions": qs})
}

// Questions godoc
//
//	@Summary	List an exam's questions (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Exam ID"
//	@Success	200	{object}	map[string][]model.Question
//	@Router		/api/v1/admin/exams/{id}/questions [get]
func (h *AdminExamHandler) Questions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	qs, err := h.exams.Questions(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, qs)
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
