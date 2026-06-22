package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractQuestionsParsesToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{"content":[{"type":"tool_use","name":"extract_questions","input":{"questions":[{"prompt":"What is your job?","sample_answer":"I'm a teacher."},{"prompt":"  "},{"prompt":"Describe your hometown.","sample_answer":""}]}}]}`))
	}))
	defer srv.Close()

	c := NewAnthropicClient("test-key")
	c.http = srv.Client()
	origURL := anthropicAPIURLOverride
	anthropicAPIURLOverride = srv.URL
	defer func() { anthropicAPIURLOverride = origURL }()

	qs, err := c.ExtractQuestions(context.Background(), "exam.txt", []byte("Q1: What is your job?"))
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 2 {
		t.Fatalf("got %d questions, want 2", len(qs))
	}
	if qs[0].Position != 1 || qs[1].Position != 2 {
		t.Fatalf("positions not sequential: %d, %d", qs[0].Position, qs[1].Position)
	}
	if qs[0].Prompt != "What is your job?" || qs[0].SampleAnswer != "I'm a teacher." {
		t.Fatalf("row 1 wrong: %+v", qs[0])
	}
}

func TestExtractQuestionsRejectsUnsupportedExt(t *testing.T) {
	c := NewAnthropicClient("test-key")
	if _, err := c.ExtractQuestions(context.Background(), "exam.docx", []byte("x")); err == nil {
		t.Fatal("want error for unsupported extension")
	}
}
