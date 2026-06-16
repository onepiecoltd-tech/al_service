package apperror

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NotFound(msg string) *AppError {
	return New(http.StatusNotFound, msg, nil)
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, msg, nil)
}

func Unauthorized(msg string) *AppError {
	return New(http.StatusUnauthorized, msg, nil)
}

func Forbidden(msg string) *AppError {
	return New(http.StatusForbidden, msg, nil)
}

func Conflict(msg string) *AppError {
	return New(http.StatusConflict, msg, nil)
}

func ServiceUnavailable(msg string) *AppError {
	return New(http.StatusServiceUnavailable, msg, nil)
}

func Internal(err error) *AppError {
	return New(http.StatusInternalServerError, "internal server error", err)
}

func StatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return http.StatusInternalServerError
}
