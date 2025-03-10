package errors

import "fmt"

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

var (
	ErrNotFound         = &AppError{Code: "NOT_FOUND", Message: "resource not found"}
	ErrUnauthorized     = &AppError{Code: "UNAUTHORIZED", Message: "unauthorized"}
	ErrInternal         = &AppError{Code: "INTERNAL", Message: "internal error"}
	ErrTooManyRedirects = fmt.Errorf("too many redirects")
	ErrInvalidURL       = fmt.Errorf("invalid URL")
	ErrFetchFailed      = fmt.Errorf("failed to fetch content")
	ErrParseFailed      = fmt.Errorf("failed to parse content")
)
