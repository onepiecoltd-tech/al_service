package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// Auth validates a Bearer JWT signed with secret and puts the user ID
// (from the "sub" claim) into the request context.
func Auth(secret string) Middleware {
	key := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			raw, ok := strings.CutPrefix(authz, "Bearer ")
			if !ok {
				httputil.Error(w, apperror.Unauthorized("missing bearer token"))
				return
			}

			token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return key, nil
			})
			if err != nil || !token.Valid {
				httputil.Error(w, apperror.Unauthorized("invalid or expired token"))
				return
			}

			sub, err := token.Claims.GetSubject()
			if err != nil {
				httputil.Error(w, apperror.Unauthorized("invalid token"))
				return
			}
			id, err := uuid.Parse(sub)
			if err != nil {
				httputil.Error(w, apperror.Unauthorized("invalid token"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user ID set by Auth.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// AdminOnly rejects requests whose authenticated user is not an admin/mod.
// Must run after Auth. isAdmin is supplied by the service layer.
func AdminOnly(isAdmin func(context.Context, uuid.UUID) (bool, error)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := UserIDFromContext(r.Context())
			if !ok {
				httputil.Error(w, apperror.Unauthorized("not authenticated"))
				return
			}
			allowed, err := isAdmin(r.Context(), id)
			if err != nil {
				httputil.Error(w, err)
				return
			}
			if !allowed {
				httputil.Error(w, apperror.Forbidden("admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// CORS allows any origin to call the API and short-circuits preflight OPTIONS
// requests. Credentials are not allowed (incompatible with a "*" origin); the
// auth token travels in the body / Authorization header, not a cookie.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start),
			"request_id", r.Context().Value(requestIDKey),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
