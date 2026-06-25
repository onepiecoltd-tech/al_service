package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/config"
	"github.com/craftbyte/learning_languages/services/internal/service"
)

type Server struct {
	cfg       *config.Config
	db        *pgxpool.Pool
	http      *http.Server
	answerJob *service.AnswerBackfiller
}

func New(cfg *config.Config, db *pgxpool.Pool) *Server {
	s := &Server{cfg: cfg, db: db}
	s.http = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	// Background AI answer-backfill, tied to the server lifecycle. Skipped when
	// no Gemini key is configured (every generate would just fail).
	if s.answerJob != nil && s.cfg.GeminiAPIKey != "" {
		go s.answerJob.Run(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.http.Shutdown(shutCtx)
	}
}
