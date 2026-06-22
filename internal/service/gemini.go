package service

import (
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

// geminiAPIURLOverride lets tests point the client at a local httptest server.
var geminiAPIURLOverride = ""

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
				"properties": map[string]any{
					"prompt":        map[string]any{"type": "STRING"},
					"sample_answer": map[string]any{"type": "STRING"},
				},
				"required": []string{"prompt"},
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
			"text": "Đây là một đề thi. Hãy đọc toàn bộ nội dung và trích ra từng câu hỏi cùng đáp án mẫu nếu đề có sẵn. Bỏ qua tiêu đề, hướng dẫn làm bài, số trang.",
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

func fileExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}
