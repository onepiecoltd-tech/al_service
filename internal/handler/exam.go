package handler

import (
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

// ExamHandler serves the user-facing exam endpoints (a user's own uploaded
// exams), as opposed to AdminExamHandler which manages the global bank.
type ExamHandler struct {
	exams    service.ExamService
	profiles service.ProfileService
}

func NewExamHandler(exams service.ExamService, profiles service.ProfileService) *ExamHandler {
	return &ExamHandler{exams: exams, profiles: profiles}
}

// Mine godoc
//
//	@Summary	List the current user's uploaded exams
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/v1/exams/mine [get]
func (h *ExamHandler) Mine(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	page, limit, offset := httputil.PageParams(r)
	exams, total, err := h.exams.ListMine(r.Context(), uid, limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Paginated(w, exams, page, limit, total)
}

// Get godoc
//
//	@Summary	Get one of the current user's uploaded exams, with its questions
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Exam ID"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/exams/{id} [get]
func (h *ExamHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	exam, err := h.exams.GetOwned(r.Context(), id, uid)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	qs, err := h.exams.Questions(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"exam": exam, "questions": qs})
}

// Upload godoc
//
//	@Summary	Upload an exam file and import its questions via AI (current user)
//	@Tags		exams
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		name	formData	string	false	"Exam name (defaults to file name)"
//	@Param		file	formData	file	true	"Exam file (.pdf or .txt)"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	errorEnvelope
//	@Router		/api/v1/exams/upload [post]
func (h *ExamHandler) Upload(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	// AI extraction can take 1-2+ minutes for large PDFs — extend past the
	// server's default 30s WriteTimeout for this handler only.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(4 * time.Minute))
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

	author := "Bạn"
	if u, err := h.profiles.Get(r.Context(), uid); err == nil {
		author = u.DisplayName
	}

	exam, qs, err := h.exams.Upload(r.Context(), uid, author, r.FormValue("name"), header.Filename, data)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"exam": exam, "imported": len(qs)})
}
