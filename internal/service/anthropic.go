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

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// anthropicAPIURLOverride lets tests point the client at a local httptest server.
var anthropicAPIURLOverride = ""

// AnthropicClient calls Claude to extract exam questions from an uploaded file.
type AnthropicClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		model:  "claude-haiku-4-5-20251001",
		http:   &http.Client{Timeout: 3 * time.Minute},
	}
}

var extractQuestionsTool = map[string]any{
	"name":        "extract_questions",
	"description": "Trả về danh sách câu hỏi và đáp án mẫu (nếu có) được trích từ đề thi.",
	"input_schema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"prompt":        map[string]any{"type": "string"},
						"sample_answer": map[string]any{"type": "string"},
					},
					"required": []string{"prompt"},
				},
			},
		},
		"required": []string{"questions"},
	},
}

// ExtractQuestions sends the exam file to Claude and returns the parsed questions.
// Supports .pdf (sent as a document content block) and .txt (sent as plain text).
func (c *AnthropicClient) ExtractQuestions(ctx context.Context, filename string, data []byte) ([]model.Question, error) {
	if c.apiKey == "" {
		return nil, apperror.Internal(fmt.Errorf("thiếu ANTHROPIC_API_KEY trên server"))
	}

	content := []map[string]any{
		{
			"type": "text",
			"text": "Đây là một đề thi. Hãy đọc toàn bộ nội dung và trích ra từng câu hỏi cùng đáp án mẫu nếu đề có sẵn. Bỏ qua tiêu đề, hướng dẫn làm bài, số trang. Gọi tool extract_questions với kết quả.",
		},
	}

	switch ext := strings.ToLower(fileExt(filename)); ext {
	case ".pdf":
		content = append(content, map[string]any{
			"type": "document",
			"source": map[string]any{
				"type":       "base64",
				"media_type": "application/pdf",
				"data":       base64.StdEncoding.EncodeToString(data),
			},
		})
	case ".txt":
		content = append(content, map[string]any{"type": "text", "text": string(data)})
	default:
		return nil, apperror.BadRequest("chỉ hỗ trợ tệp .pdf hoặc .txt")
	}

	reqBody := map[string]any{
		"model":       c.model,
		"max_tokens":  8192,
		"tools":       []any{extractQuestionsTool},
		"tool_choice": map[string]any{"type": "tool", "name": "extract_questions"},
		"messages": []map[string]any{
			{"role": "user", "content": content},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	url := anthropicAPIURL
	if anthropicAPIURLOverride != "" {
		url = anthropicAPIURLOverride
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("gọi Claude API thất bại: %w", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apperror.Internal(fmt.Errorf("Claude API lỗi (%d): %s", resp.StatusCode, string(respBody)))
	}

	var parsed struct {
		Content []struct {
			Type  string          `json:"type"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, apperror.Internal(fmt.Errorf("không đọc được phản hồi từ Claude: %w", err))
	}

	for _, block := range parsed.Content {
		if block.Type != "tool_use" {
			continue
		}
		var out struct {
			Questions []struct {
				Prompt       string `json:"prompt"`
				SampleAnswer string `json:"sample_answer"`
			} `json:"questions"`
		}
		if err := json.Unmarshal(block.Input, &out); err != nil {
			return nil, apperror.Internal(fmt.Errorf("không đọc được câu hỏi từ Claude: %w", err))
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
	return nil, apperror.Internal(fmt.Errorf("Claude không trả về kết quả trích xuất"))
}

func fileExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}
