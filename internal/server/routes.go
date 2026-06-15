package server

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/middleware"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)

	// Register domain routes here, e.g.:
	// mux.Handle("/api/v1/", userHandler.Routes())

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger,
	)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
