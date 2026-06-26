package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	exams, total, err := h.exams.ListMine(r.Context(), uid, r.URL.Query().Get("lang"), limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Paginated(w, exams, page, limit, total)
}

// PracticeQuestions godoc
//
//	@Summary	Pool questions of a skill across exams, for skill-based practice
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Param		skill	query		string	true	"Skill: listening|reading|writing|speaking"
//	@Param		lang	query		string	false	"Language code filter"
//	@Param		source	query		string	false	"Source filter: bank|mine (default both)"
//	@Param		q		query		string	false	"Full-text search over prompt and sample answer"
//	@Param		limit	query		int		false	"Max questions (default 10)"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	errorEnvelope
//	@Router		/api/v1/questions [get]
func (h *ExamHandler) PracticeQuestions(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	qs, err := h.exams.PracticeQuestions(r.Context(), uid, r.URL.Query().Get("skill"), r.URL.Query().Get("lang"), r.URL.Query().Get("source"), r.URL.Query().Get("q"), limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"questions": qs})
}

// Bank godoc
//
//	@Summary	List the published admin question bank, for any user to practice
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Router		/api/v1/exam-bank [get]
func (h *ExamHandler) Bank(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := httputil.PageParams(r)
	exams, total, err := h.exams.ListBank(r.Context(), r.URL.Query().Get("lang"), limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Paginated(w, exams, page, limit, total)
}

// RandomBankQuestion godoc
//
//	@Summary	Get one random question from the published bank, for a random speaking/practice prompt
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/exam-bank/random-question [get]
func (h *ExamHandler) RandomBankQuestion(w http.ResponseWriter, r *http.Request) {
	q, err := h.exams.RandomBankQuestion(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, q)
}

// BankGet godoc
//
//	@Summary	Get one published bank exam, with its questions
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Exam ID"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/exam-bank/{id} [get]
func (h *ExamHandler) BankGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	exam, err := h.exams.GetBank(r.Context(), id)
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

// Ask godoc
//
//	@Summary	Ask the AI tutor a question about one of the current user's exams
//	@Description	Streams the answer as it's generated via Server-Sent Events (one "data:" JSON event per text fragment, ending with {"done":true}), and persists both the question and the full answer to the exam's chat history. GET + query param (not POST + body) because the client consumes this via the native EventSource API, which only issues GET requests.
//	@Tags		exams
//	@Produce	text/event-stream
//	@Security	BearerAuth
//	@Param		id			path	string	true	"Exam ID"
//	@Param		question	query	string	true	"Question"
//	@Failure	400			{object}	errorEnvelope
//	@Failure	404			{object}	errorEnvelope
//	@Router		/api/v1/exams/{id}/ask [get]
func (h *ExamHandler) Ask(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	question := strings.TrimSpace(r.URL.Query().Get("question"))
	if question == "" {
		httputil.Error(w, apperror.BadRequest("thiếu câu hỏi"))
		return
	}
	// Ownership is checked up front, before any SSE headers go out, so a
	// rejected request still gets a normal JSON error response.
	if _, err := h.exams.GetOwned(r.Context(), id, uid); err != nil {
		httputil.Error(w, err)
		return
	}

	// Gemini calls can take tens of seconds; extend past the server's default
	// 30s WriteTimeout for this handler only.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(2 * time.Minute))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering, if any sits in front
	w.WriteHeader(http.StatusOK)

	writeEvent := func(payload map[string]any) {
		buf, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", buf)
		_ = rc.Flush()
	}

	err = h.exams.AskStream(r.Context(), id, uid, question, func(chunk string) {
		writeEvent(map[string]any{"text": chunk})
	})
	if err != nil {
		writeEvent(map[string]any{"error": err.Error()})
		return
	}
	writeEvent(map[string]any{"done": true})
}

// ChatHistory godoc
//
//	@Summary	Get the persisted Giải đề AI conversation for one of the current user's exams
//	@Tags		exams
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Exam ID"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/exams/{id}/chat [get]
func (h *ExamHandler) ChatHistory(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid exam id"))
		return
	}
	msgs, err := h.exams.ChatHistory(r.Context(), id, uid)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"messages": msgs})
}

// Upload godoc
//
//	@Summary	Upload an exam file and import its questions via AI (current user)
//	@Tags		exams
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		name		formData	string	false	"Exam name (defaults to file name)"
//	@Param		language	formData	string	false	"Target language code (defaults to en)"
//	@Param		file		formData	file	true	"Exam file (.pdf or .txt)"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	errorEnvelope
//	@Router		/api/v1/exams/upload [post]
func (h *ExamHandler) Upload(w http.ResponseWriter, r *http.Request) {
	uid, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
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

	exam, err := h.exams.CreateUpload(r.Context(), uid, author, r.FormValue("name"), r.FormValue("language"), header.Filename)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	// Extract questions in the background so the client gets a fast response and
	// can show "uploaded, AI is extracting" instead of waiting minutes. The
	// goroutine outlives the request, so it uses a fresh context, not r.Context().
	filename := header.Filename
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
		defer cancel()
		h.exams.ExtractUpload(ctx, exam.ID, filename, data)
	}()

	httputil.OK(w, map[string]any{"exam": exam, "processing": true})
}
