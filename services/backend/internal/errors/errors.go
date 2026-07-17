package errors

import (
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Wrap(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: msg,
		Err:     e,
	}
}

func New(message string, code ...string) *AppError {
	err := &AppError{
		Message: message,
	}
	if len(code) > 0 {
		err.Code = code[0]
	}
	return err
}

func IsNotFound(err error) bool {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "NOT_FOUND"
	}
	return false
}

func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "UNAUTHORIZED"
	}
	return false
}

// StatusCode maps an error to the HTTP status a handler should return for it. Known cases
// (not-found, unauthorized/cross-tenant) get their specific status; anything else — including
// raw GORM/driver errors that must never reach the client — falls back to 500 so callers know to
// log the real error and return a generic message instead of the error's own text.
func StatusCode(err error) int {
	switch {
	case IsNotFound(err):
		return http.StatusNotFound
	case IsUnauthorized(err):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

var (
	ErrNotFound         = &AppError{Code: "NOT_FOUND", Message: "resource not found"}
	ErrUnauthorized     = &AppError{Code: "UNAUTHORIZED", Message: "unauthorized"}
	ErrInternal         = &AppError{Code: "INTERNAL", Message: "internal error"}
	ErrTooManyRedirects = fmt.Errorf("too many redirects")
	ErrInvalidURL       = fmt.Errorf("invalid URL")
	ErrFetchFailed      = fmt.Errorf("failed to fetch content")
	ErrParseFailed      = fmt.Errorf("failed to parse content")
)
