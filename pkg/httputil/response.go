package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
)

type envelope map[string]any

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
