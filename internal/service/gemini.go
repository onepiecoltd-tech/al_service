package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

// pdfConfig builds a lenient pdfcpu config for splitting exam PDFs: relaxed
// validation (the default) plus the optimization passes disabled, since those
// are the usual source of crashes/panics on complex, real-world PDFs.
func pdfConfig() *pdfmodel.Configuration {
	conf := pdfmodel.NewDefaultConfiguration()
	conf.ValidationMode = pdfmodel.ValidationRelaxed
	conf.Optimize = false
	conf.OptimizeBeforeWriting = false
	return conf
}

const geminiAPIURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
const geminiStreamAPIURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse"
const geminiUploadURL = "https://generativelanguage.googleapis.com/upload/v1beta/files"
const geminiFileURLTemplate = "https://generativelanguage.googleapis.com/v1beta/%s" // %s = file resource name, e.g. "files/abc123"

// geminiAPIURLOverride lets tests point the client at a local httptest server.
var geminiAPIURLOverride = ""

// geminiStreamAPIURLOverride is the streaming-endpoint equivalent of geminiAPIURLOverride.
var geminiStreamAPIURLOverride = ""

// GeminiClient calls Gemini to extract exam questions from an uploaded file.
type GeminiClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey: apiKey,
		model:  "gemini-2.5-flash",
		// Generous backstop only — per-request context deadlines (e.g. the
		// background extraction context) are the real governor. Extracting a
		// large exam PDF can take several minutes, well past the old 3-minute cap.
		http: &http.Client{Timeout: 10 * time.Minute},
	}
}

var extractQuestionsSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"questions": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				// Descriptions tell the model exactly which text goes where —
				// without them it guesses and often swaps question and answer.
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "STRING",
						"description": "Nội dung câu hỏi / đề bài mà người làm bài phải trả lời. Tuyệt đối KHÔNG đặt đáp án vào đây.",
					},
					"sample_answer": map[string]any{
						"type":        "STRING",
						"description": "Đáp án mẫu cho câu hỏi, CHỈ khi đề có sẵn đáp án; nếu không có thì để trống.",
					},
				},
				"propertyOrdering": []string{"prompt", "sample_answer"},
				"required":         []string{"prompt"},
			},
		},
	},
	"required": []string{"questions"},
}

// extractInstruction tells Gemini what to pull out of each exam page/chunk.
const extractInstruction = "Đây là một đề thi. Hãy đọc toàn bộ nội dung và trích ra từng câu hỏi. " +
	"Với mỗi mục: 'prompt' là nội dung câu hỏi/đề bài (KHÔNG bao giờ chứa đáp án), " +
	"'sample_answer' là đáp án mẫu CHỈ khi đề có sẵn, không có thì để trống. " +
	"Nếu một mục không có phần câu hỏi (ví dụ chỉ là bảng đáp án) thì bỏ qua, " +
	"tuyệt đối không tự bịa câu hỏi và không chuyển đáp án vào ô câu hỏi. " +
	"Bỏ qua tiêu đề, hướng dẫn làm bài, số trang."

// A big exam book is split into page-range chunks so no single Gemini call is
// too slow or has its JSON output truncated; chunks extract in parallel.
const pagesPerChunk = 10
const chunkConcurrency = 3

type parsedQuestion struct {
	Prompt       string `json:"prompt"`
	SampleAnswer string `json:"sample_answer"`
}

// ExtractQuestions extracts exam questions from an uploaded file. .txt is sent
// inline; .pdf is uploaded via the Files API and, when long, split into
// page-range chunks extracted in parallel and merged in page order.
func (c *GeminiClient) ExtractQuestions(ctx context.Context, filename string, data []byte) ([]model.Question, error) {
	if c.apiKey == "" {
		return nil, apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	var raw []parsedQuestion
	switch ext := strings.ToLower(fileExt(filename)); ext {
	case ".pdf":
		chunks, err := splitPDF(data, pagesPerChunk)
		if err != nil {
			// Some PDFs can't be parsed/split by pdfcpu but Gemini may still read
			// them — fall back to extracting the whole file as one chunk.
			slog.Warn("pdf split failed, extracting whole file", "filename", filename, "error", err)
			chunks = [][]byte{data}
		} else {
			slog.Info("pdf split for extraction", "filename", filename, "chunks", len(chunks))
		}
		raw, err = c.extractPDFChunks(ctx, chunks)
		if err != nil {
			return nil, err
		}
	case ".txt":
		var err error
		raw, err = c.generateQuestions(ctx, []map[string]any{
			{"text": extractInstruction},
			{"text": string(data)},
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, apperror.BadRequest("chỉ hỗ trợ tệp .pdf hoặc .txt")
	}

	qs := make([]model.Question, 0, len(raw))
	for _, q := range raw {
		prompt := strings.TrimSpace(q.Prompt)
		if prompt == "" {
			continue
		}
		qs = append(qs, model.Question{
			Position:     len(qs) + 1,
			Prompt:       prompt,
			SampleAnswer: strings.TrimSpace(q.SampleAnswer),
		})
	}
	if len(qs) == 0 {
		return nil, apperror.BadRequest("AI không tìm thấy câu hỏi nào trong tệp")
	}
	return qs, nil
}

// generateQuestions runs a single generateContent call with the given parts and
// returns the raw (unfiltered, unpositioned) question list.
func (c *GeminiClient) generateQuestions(ctx context.Context, parts []map[string]any) ([]parsedQuestion, error) {
	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": parts},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   extractQuestionsSchema,
			// A full exam can yield a long question list — raise the output cap
			// so the JSON isn't truncated mid-array (which fails parsing).
			"maxOutputTokens": 65536,
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	url := fmt.Sprintf(geminiAPIURLTemplate, c.model)
	if geminiAPIURLOverride != "" {
		url = geminiAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("gọi Gemini API thất bại: %w", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apperror.Internal(fmt.Errorf("Gemini API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được phản hồi từ Gemini: %w", err))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		// MAX_TOKENS with no usable content means the input was too large for one
		// pass — surface a clear, actionable message instead of a vague decode error.
		if len(parsed.Candidates) > 0 && parsed.Candidates[0].FinishReason == "MAX_TOKENS" {
			return nil, apperror.BadRequest("phần đề quá lớn để AI xử lý một lần — hãy tải lên từng phần nhỏ hơn")
		}
		return nil, apperror.Internal(fmt.Errorf("Gemini không trả về kết quả trích xuất"))
	}
	if parsed.Candidates[0].FinishReason == "MAX_TOKENS" {
		return nil, apperror.BadRequest("phần đề quá lớn để AI xử lý một lần — hãy tải lên từng phần nhỏ hơn")
	}

	var out struct {
		Questions []parsedQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(parsed.Candidates[0].Content.Parts[0].Text), &out); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được câu hỏi từ Gemini: %w", err))
	}
	return out.Questions, nil
}

// GenerateAnswer produces a concise model answer for a single exam question,
// in the exam's target language. Used by the background answer-backfill job.
func (c *GeminiClient) GenerateAnswer(ctx context.Context, prompt, language string) (string, error) {
	if c.apiKey == "" {
		return "", apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	sys := "Bạn là giáo viên luyện thi. Hãy viết một đáp án mẫu ngắn gọn, chất lượng cao cho câu hỏi/đề bài sau, " +
		"bằng đúng ngôn ngữ của đề thi"
	if language != "" {
		sys += " (mã ngôn ngữ: " + language + ")"
	}
	sys += ". Chỉ trả về nội dung đáp án, không thêm lời dẫn hay giải thích thừa."

	reqBody := map[string]any{
		"system_instruction": map[string]any{"parts": []map[string]any{{"text": sys}}},
		"contents":           []map[string]any{{"parts": []map[string]any{{"text": prompt}}}},
		// A model answer is straightforward generation — skip the thinking phase
		// to keep the batch job fast and cheap.
		"generationConfig": map[string]any{"thinkingConfig": map[string]any{"thinkingBudget": 0}},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", apperror.Internal(err)
	}

	url := fmt.Sprintf(geminiAPIURLTemplate, c.model)
	if geminiAPIURLOverride != "" {
		url = geminiAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("gọi Gemini API thất bại: %w", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apperror.Internal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", apperror.Internal(fmt.Errorf("Gemini API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", apperror.Internal(fmt.Errorf("không đọc được phản hồi từ Gemini: %w", err))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", apperror.Internal(fmt.Errorf("Gemini không trả về đáp án"))
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// extractPDFChunks uploads each page-range chunk to the Files API and extracts
// its questions with bounded parallelism, then concatenates them in page order.
// A failed chunk is logged and skipped; only an all-chunks failure is an error.
func (c *GeminiClient) extractPDFChunks(ctx context.Context, chunks [][]byte) ([]parsedQuestion, error) {
	results := make([][]parsedQuestion, len(chunks))
	errs := make([]error, len(chunks))
	sem := make(chan struct{}, chunkConcurrency)
	var wg sync.WaitGroup
	for i := range chunks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// A panic in a worker goroutine would crash the whole process; turn
			// it into a chunk-level error instead.
			defer func() {
				if r := recover(); r != nil {
					errs[i] = fmt.Errorf("panic khi xử lý phần %d: %v", i+1, r)
				}
			}()
			sem <- struct{}{}
			defer func() { <-sem }()

			uri, err := c.uploadFile(ctx, "application/pdf", fmt.Sprintf("exam-chunk-%d.pdf", i+1), chunks[i])
			if err != nil {
				errs[i] = err
				return
			}
			qs, err := c.generateQuestions(ctx, []map[string]any{
				{"text": extractInstruction},
				{"file_data": map[string]any{"mime_type": "application/pdf", "file_uri": uri}},
			})
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = qs
		}(i)
	}
	wg.Wait()

	failed := 0
	var firstErr error
	for i, e := range errs {
		if e != nil {
			failed++
			if firstErr == nil {
				firstErr = e
			}
			slog.Error("exam chunk extraction failed", "chunk", i+1, "of", len(chunks), "error", e)
		}
	}
	if failed == len(chunks) {
		return nil, firstErr
	}

	var all []parsedQuestion
	for _, r := range results {
		all = append(all, r...)
	}
	return all, nil
}

// splitPDF splits a PDF into chunks of at most perChunk pages each. A PDF with
// no more than perChunk pages is returned unchanged as a single chunk.
func splitPDF(data []byte, perChunk int) (chunks [][]byte, err error) {
	// pdfcpu can panic (nil deref) on some malformed PDFs and re-panic past its
	// own recovery — convert that to an error so the caller falls back to
	// whole-file extraction rather than crashing the process.
	defer func() {
		if r := recover(); r != nil {
			chunks, err = nil, fmt.Errorf("pdfcpu panic: %v", r)
		}
	}()

	conf := pdfConfig()
	n, err := pdfapi.PageCount(bytes.NewReader(data), conf)
	if err != nil {
		return nil, err
	}
	if n <= perChunk {
		return [][]byte{data}, nil
	}
	for start := 1; start <= n; start += perChunk {
		end := start + perChunk - 1
		if end > n {
			end = n
		}
		var buf bytes.Buffer
		if err := pdfapi.Trim(bytes.NewReader(data), &buf, []string{fmt.Sprintf("%d-%d", start, end)}, pdfConfig()); err != nil {
			return nil, err
		}
		chunks = append(chunks, buf.Bytes())
	}
	return chunks, nil
}

// uploadFile uploads a file to the Gemini Files API via the resumable protocol
// and waits until it becomes ACTIVE, returning its file URI for use in a
// file_data part. Used for PDFs, which can exceed the inline_data request limit.
func (c *GeminiClient) uploadFile(ctx context.Context, mimeType, displayName string, data []byte) (string, error) {
	// 1. Start a resumable upload session; Gemini replies with an upload URL.
	startBody, _ := json.Marshal(map[string]any{"file": map[string]any{"display_name": displayName}})
	startReq, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiUploadURL, bytes.NewReader(startBody))
	if err != nil {
		return "", apperror.Internal(err)
	}
	startReq.Header.Set("x-goog-api-key", c.apiKey)
	startReq.Header.Set("X-Goog-Upload-Protocol", "resumable")
	startReq.Header.Set("X-Goog-Upload-Command", "start")
	startReq.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.Itoa(len(data)))
	startReq.Header.Set("X-Goog-Upload-Header-Content-Type", mimeType)
	startReq.Header.Set("content-type", "application/json")
	startResp, err := c.http.Do(startReq)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("bắt đầu tải tệp lên Gemini thất bại: %w", err))
	}
	uploadURL := startResp.Header.Get("X-Goog-Upload-URL")
	startResp.Body.Close()
	if startResp.StatusCode != http.StatusOK || uploadURL == "" {
		return "", apperror.Internal(fmt.Errorf("Gemini không cấp URL tải tệp (%d)", startResp.StatusCode))
	}

	// 2. Upload all the bytes in one request and finalize.
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", apperror.Internal(err)
	}
	upReq.Header.Set("x-goog-api-key", c.apiKey)
	upReq.Header.Set("X-Goog-Upload-Offset", "0")
	upReq.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	upResp, err := c.http.Do(upReq)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("tải tệp lên Gemini thất bại: %w", err))
	}
	defer upResp.Body.Close()
	upBody, _ := io.ReadAll(upResp.Body)
	if upResp.StatusCode != http.StatusOK {
		return "", apperror.Internal(fmt.Errorf("Gemini tải tệp lỗi (%d): %s", upResp.StatusCode, string(upBody)))
	}
	var fr struct {
		File struct {
			Name  string `json:"name"`
			URI   string `json:"uri"`
			State string `json:"state"`
		} `json:"file"`
	}
	if err := json.Unmarshal(upBody, &fr); err != nil {
		return "", apperror.Internal(fmt.Errorf("không đọc được phản hồi tải tệp: %w", err))
	}
	if fr.File.URI == "" || fr.File.Name == "" {
		return "", apperror.Internal(fmt.Errorf("Gemini không trả về thông tin tệp"))
	}

	// 3. A freshly uploaded PDF is PROCESSING for a moment; wait until ACTIVE.
	state := fr.File.State
	for state == "PROCESSING" || state == "" {
		select {
		case <-ctx.Done():
			return "", apperror.Internal(ctx.Err())
		case <-time.After(2 * time.Second):
		}
		if state, err = c.fileState(ctx, fr.File.Name); err != nil {
			return "", err
		}
	}
	if state != "ACTIVE" {
		return "", apperror.BadRequest("Gemini không xử lý được tệp đề thi")
	}
	return fr.File.URI, nil
}

// fileState returns the processing state of an uploaded Files API file.
func (c *GeminiClient) fileState(ctx context.Context, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(geminiFileURLTemplate, name), nil)
	if err != nil {
		return "", apperror.Internal(err)
	}
	req.Header.Set("x-goog-api-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", apperror.Internal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", apperror.Internal(fmt.Errorf("Gemini kiểm tra tệp lỗi (%d): %s", resp.StatusCode, string(body)))
	}
	var f struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(body, &f); err != nil {
		return "", apperror.Internal(err)
	}
	return f.State, nil
}

// ChatTurn is one message in a Giải đề AI conversation.
type ChatTurn struct {
	Role string // "user" or "model"
	Text string
}

// AskStream answers a free-text question about an exam's questions, given the
// prior conversation turns, streaming each text fragment to onChunk as Gemini
// produces it. Returns the full concatenated answer once the stream ends.
func (c *GeminiClient) AskStream(ctx context.Context, examContext string, history []ChatTurn, question string, onChunk func(chunk string)) (string, error) {
	if c.apiKey == "" {
		return "", apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	contents := make([]map[string]any, 0, len(history)+1)
	for _, t := range history {
		contents = append(contents, map[string]any{
			"role":  t.Role,
			"parts": []map[string]any{{"text": t.Text}},
		})
	}
	contents = append(contents, map[string]any{
		"role":  "user",
		"parts": []map[string]any{{"text": question}},
	})

	reqBody := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]any{{"text": "Bạn là trợ lý giải đề thi cho học viên. Dưới đây là các câu hỏi (và đáp án mẫu nếu có) của đề thi đang được hỏi:\n\n" + examContext + "\n\nHãy trả lời ngắn gọn, rõ ràng, bằng tiếng Việt, tập trung vào câu hỏi của học viên."}},
		},
		"contents": contents,
		// This is a quick tutoring Q&A, not a task needing deep reasoning —
		// disabling the "thinking" phase removes a long silent gap before any
		// output and makes the stream emit fragments sooner/more granularly.
		"generationConfig": map[string]any{
			"thinkingConfig": map[string]any{"thinkingBudget": 0},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", apperror.Internal(err)
	}

	url := fmt.Sprintf(geminiStreamAPIURLTemplate, c.model)
	if geminiStreamAPIURLOverride != "" {
		url = geminiStreamAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("gọi Gemini API thất bại: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", apperror.Internal(fmt.Errorf("Gemini API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed/keepalive lines rather than aborting the whole answer
		}
		for _, cand := range chunk.Candidates {
			for _, p := range cand.Content.Parts {
				if p.Text == "" {
					continue
				}
				full.WriteString(p.Text)
				onChunk(p.Text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", apperror.Internal(fmt.Errorf("đọc luồng phản hồi từ Gemini thất bại: %w", err))
	}

	answer := strings.TrimSpace(full.String())
	if answer == "" {
		return "", apperror.Internal(fmt.Errorf("Gemini trả về câu trả lời trống"))
	}
	return answer, nil
}

var speakingSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"band_overall":  map[string]any{"type": "NUMBER"},
		"fluency":       map[string]any{"type": "NUMBER"},
		"vocabulary":    map[string]any{"type": "NUMBER"},
		"grammar":       map[string]any{"type": "NUMBER"},
		"pronunciation": map[string]any{"type": "NUMBER"},
		"feedback":      map[string]any{"type": "STRING"},
	},
	"required": []string{"band_overall", "fluency", "vocabulary", "grammar", "pronunciation", "feedback"},
}

// SpeakingResult is the AI's assessment of one spoken answer.
type SpeakingResult struct {
	BandOverall   float64 `json:"band_overall"`
	Fluency       float64 `json:"fluency"`
	Vocabulary    float64 `json:"vocabulary"`
	Grammar       float64 `json:"grammar"`
	Pronunciation float64 `json:"pronunciation"`
	Feedback      string  `json:"feedback"`
}

// GradeSpeaking sends a recorded spoken answer to Gemini and returns an
// IELTS-style band assessment. mimeType should match the audio encoding the
// browser recorded with (e.g. "audio/webm").
func (c *GeminiClient) GradeSpeaking(ctx context.Context, promptText, mimeType string, audio []byte) (*SpeakingResult, error) {
	if c.apiKey == "" {
		return nil, apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	parts := []map[string]any{
		{
			"text": "Bạn là giám khảo IELTS Speaking. Đây là câu hỏi/đề bài và bản ghi âm câu trả lời của học viên. Hãy chấm điểm theo 4 tiêu chí (thang 0-9, có thể lẻ 0.5) và viết nhận xét ngắn gọn bằng tiếng Việt (3-5 câu), chỉ ra điểm mạnh và điều cần cải thiện cụ thể.\n\nĐề bài: " + promptText,
		},
		{
			"inline_data": map[string]any{
				"mime_type": mimeType,
				"data":      base64.StdEncoding.EncodeToString(audio),
			},
		},
	}

	reqBody := map[string]any{
		"contents": []map[string]any{{"parts": parts}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   speakingSchema,
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	url := fmt.Sprintf(geminiAPIURLTemplate, c.model)
	if geminiAPIURLOverride != "" {
		url = geminiAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("gọi Gemini API thất bại: %w", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apperror.Internal(fmt.Errorf("Gemini API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được phản hồi từ Gemini: %w", err))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, apperror.Internal(fmt.Errorf("Gemini không trả về kết quả chấm điểm"))
	}

	var out SpeakingResult
	if err := json.Unmarshal([]byte(parsed.Candidates[0].Content.Parts[0].Text), &out); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được điểm số từ Gemini: %w", err))
	}
	return &out, nil
}

// TranscribePhonetic asks Gemini for the IPA phonetic transcription of a
// single English word, e.g. "/ˌɒn.trə.prəˈnɜːr/".
func (c *GeminiClient) TranscribePhonetic(ctx context.Context, word string) (string, error) {
	if c.apiKey == "" {
		return "", apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": "Give the IPA phonetic transcription (General American or RP, your choice) of this English word, wrapped in slashes, e.g. /wɜːd/. Reply with ONLY the transcription, nothing else.\n\nWord: " + word}}},
		},
		"generationConfig": map[string]any{
			"thinkingConfig": map[string]any{"thinkingBudget": 0},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", apperror.Internal(err)
	}

	url := fmt.Sprintf(geminiAPIURLTemplate, c.model)
	if geminiAPIURLOverride != "" {
		url = geminiAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", apperror.Internal(fmt.Errorf("gọi Gemini API thất bại: %w", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apperror.Internal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", apperror.Internal(fmt.Errorf("Gemini API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", apperror.Internal(fmt.Errorf("không đọc được phản hồi từ Gemini: %w", err))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", apperror.Internal(fmt.Errorf("Gemini không trả về kết quả"))
	}
	return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), nil
}

func fileExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}
