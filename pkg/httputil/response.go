package httputil

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
)

type envelope map[string]any

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PageParams parses ?page & ?limit (defaults 1 / 20, limit capped at 100) and
// returns the SQL offset.
func PageParams(r *http.Request) (page, limit, offset int) {
	page = atoiOr(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	limit = atoiOr(r.URL.Query().Get("limit"), 20)
	if limit < 1 {
		limit = 20
	}
	limit = min(limit, 100)
	return page, limit, (page - 1) * limit
}

// Paginated writes { data, meta } with the standard pagination envelope.
func Paginated(w http.ResponseWriter, data any, page, limit, total int) {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	JSON(w, http.StatusOK, envelope{
		"data": data,
		"meta": Meta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	})
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, envelope{"data": data})
}

func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, envelope{"data": data})
}

func Error(w http.ResponseWriter, err error) {
	code := apperror.StatusCode(err)
	JSON(w, code, envelope{"error": err.Error()})
}
