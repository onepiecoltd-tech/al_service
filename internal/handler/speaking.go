package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type SpeakingHandler struct {
	speaking service.SpeakingService
}

func NewSpeakingHandler(speaking service.SpeakingService) *SpeakingHandler {
	return &SpeakingHandler{speaking: speaking}
}

// Grade godoc
//
//	@Summary	Grade a recorded spoken answer via AI (IELTS-style band scores)
//	@Tags		speaking
//	@Accept		mpfd
//	@Produce	json
//	@Security	BearerAuth
//	@Param		prompt	formData	string	true	"Speaking prompt/cue the user answered"
//	@Param		audio	formData	file	true	"Recorded audio (webm/wav/mp3)"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	400	{object}	errorEnvelope
//	@Router		/api/v1/speaking/grade [post]
func (h *SpeakingHandler) Grade(w http.ResponseWriter, r *http.Request) {
	// Gemini audio grading can take tens of seconds — extend past the
	// server's default 30s WriteTimeout for this handler only.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20) // 15 MB cap — short clips only

	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		httputil.Error(w, apperror.BadRequest("thiếu đề bài"))
		return
	}
	file, header, err := r.FormFile("audio")
	if err != nil {
		if err.Error() == "http: request body too large" {
			httputil.Error(w, apperror.BadRequest("tệp ghi âm vượt quá giới hạn 15MB"))
			return
		}
		httputil.Error(w, apperror.BadRequest("thiếu tệp ghi âm (field \"audio\")"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		httputil.Error(w, apperror.BadRequest("không đọc được tệp ghi âm"))
		return
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "audio/webm"
	}

	result, err := h.speaking.Grade(r.Context(), prompt, mimeType, data)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, result)
}

// PracticeWord godoc
//
//	@Summary	Get a word for the pronunciation drill by name, inserting it if new
//	@Tags		speaking
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		object{word=string}	true	"Word to practice"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	errorEnvelope
//	@Router		/api/v1/speaking/word [post]
func (h *SpeakingHandler) PracticeWord(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Word string `json:"word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.Error(w, apperror.BadRequest("dữ liệu không hợp lệ"))
		return
	}
	word, err := h.speaking.PracticeWord(r.Context(), body.Word)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, word)
}

// RandomWord godoc
//
//	@Summary	Get one random word for the pronunciation drill
//	@Tags		speaking
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	map[string]interface{}
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/speaking/random-word [get]
func (h *SpeakingHandler) RandomWord(w http.ResponseWriter, r *http.Request) {
	word, err := h.speaking.RandomWord(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, word)
}
