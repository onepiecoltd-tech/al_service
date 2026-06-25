package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/craftbyte/learning_languages/services/internal/repository"
)

// AnswerBackfiller periodically fills in missing sample answers for questions,
// generating them with the AI. It works in small batches over a partial-indexed
// query, so it never scans the whole questions table.
type AnswerBackfiller struct {
	questions repository.QuestionRepository
	ai        *GeminiClient
	interval  time.Duration
	batch     int
	mu        sync.Mutex // guards against overlapping runs (ticker vs manual trigger)
}

func NewAnswerBackfiller(questions repository.QuestionRepository, ai *GeminiClient) *AnswerBackfiller {
	return &AnswerBackfiller{
		questions: questions,
		ai:        ai,
		interval:  time.Hour,
		batch:     20,
	}
}

// Run processes a batch immediately, then once per interval until ctx is done.
func (b *AnswerBackfiller) Run(ctx context.Context) {
	t := time.NewTicker(b.interval)
	defer t.Stop()
	for {
		b.guardedBatch(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Trigger kicks off a backfill in the background that drains the questions
// currently missing an answer, unless a run is already in progress. Returns
// false if one is already running (so the caller can say "đang chạy"). It uses
// its own context, since it outlives the HTTP request that triggered it.
func (b *AnswerBackfiller) Trigger() bool {
	if !b.mu.TryLock() {
		return false
	}
	go func() {
		defer b.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		for ctx.Err() == nil {
			if b.runBatch(ctx) == 0 {
				break
			}
		}
	}()
	return true
}

// guardedBatch runs one batch unless a run is already in progress.
func (b *AnswerBackfiller) guardedBatch(ctx context.Context) {
	if !b.mu.TryLock() {
		return
	}
	defer b.mu.Unlock()
	b.runBatch(ctx)
}

// runBatch processes one batch and returns how many questions it fetched (0 when
// nothing is pending, which a drain loop uses to stop). Caller holds b.mu.
func (b *AnswerBackfiller) runBatch(ctx context.Context) int {
	// This runs in a background goroutine — a panic here must not crash the server.
	defer func() {
		if r := recover(); r != nil {
			slog.Error("answer backfill batch panicked", "error", r)
		}
	}()

	qs, err := b.questions.ListMissingAnswers(ctx, b.batch)
	if err != nil {
		slog.Error("answer backfill: list failed", "error", err)
		return 0
	}
	for _, q := range qs {
		if ctx.Err() != nil {
			return 0
		}
		answer, err := b.ai.GenerateAnswer(ctx, q.Prompt, q.Language)
		if err != nil {
			slog.Warn("answer backfill: generate failed", "question_id", q.ID, "error", err)
			_ = b.questions.BumpAnswerAttempt(ctx, q.ID)
			continue
		}
		answer = strings.TrimSpace(answer)
		if answer == "" {
			_ = b.questions.BumpAnswerAttempt(ctx, q.ID)
			continue
		}
		if err := b.questions.SetSampleAnswer(ctx, q.ID, answer); err != nil {
			slog.Error("answer backfill: save failed", "question_id", q.ID, "error", err)
		}
	}
	return len(qs)
}
