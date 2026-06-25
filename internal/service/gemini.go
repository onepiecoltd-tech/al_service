package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

const geminiAPIURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
const geminiStreamAPIURLTemplate = "https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse"

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
		http:   &http.Client{Timeout: 3 * time.Minute},
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

// ExtractQuestions sends the exam file to Gemini and returns the parsed questions.
// Supports .pdf (sent as an inline_data part) and .txt (sent as plain text).
func (c *GeminiClient) ExtractQuestions(ctx context.Context, filename string, data []byte) ([]model.Question, error) {
	if c.apiKey == "" {
		return nil, apperror.Internal(fmt.Errorf("thiếu GEMINI_API_KEY trên server"))
	}

	parts := []map[string]any{
		{
			"text": "Đây là một đề thi. Hãy đọc toàn bộ nội dung và trích ra từng câu hỏi. " +
				"Với mỗi mục: 'prompt' là nội dung câu hỏi/đề bài (KHÔNG bao giờ chứa đáp án), " +
				"'sample_answer' là đáp án mẫu CHỈ khi đề có sẵn, không có thì để trống. " +
				"Nếu một mục không có phần câu hỏi (ví dụ chỉ là bảng đáp án) thì bỏ qua, " +
				"tuyệt đối không tự bịa câu hỏi và không chuyển đáp án vào ô câu hỏi. " +
				"Bỏ qua tiêu đề, hướng dẫn làm bài, số trang.",
		},
	}

	switch ext := strings.ToLower(fileExt(filename)); ext {
	case ".pdf":
		parts = append(parts, map[string]any{
			"inline_data": map[string]any{
				"mime_type": "application/pdf",
				"data":      base64.StdEncoding.EncodeToString(data),
			},
		})
	case ".txt":
		parts = append(parts, map[string]any{"text": string(data)})
	default:
		return nil, apperror.BadRequest("chỉ hỗ trợ tệp .pdf hoặc .txt")
	}

	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": parts},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   extractQuestionsSchema,
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
		return nil, apperror.Internal(fmt.Errorf("Gemini không trả về kết quả trích xuất"))
	}

	var out struct {
		Questions []struct {
			Prompt       string `json:"prompt"`
			SampleAnswer string `json:"sample_answer"`
		} `json:"questions"`
	}
	if err := json.Unmarshal([]byte(parsed.Candidates[0].Content.Parts[0].Text), &out); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được câu hỏi từ Gemini: %w", err))
	}

	qs := make([]model.Question, 0, len(out.Questions))
	for _, q := range out.Questions {
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
