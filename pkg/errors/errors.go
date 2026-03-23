package errors

import (
	"errors"
	"net/http"
)

type AppError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func BadRequest(message string) error {
	return &AppError{StatusCode: http.StatusBadRequest, Message: message}
}

func Unauthorized(message string) error {
	return &AppError{StatusCode: http.StatusUnauthorized, Message: message}
}

func Forbidden(message string) error {
	return &AppError{StatusCode: http.StatusForbidden, Message: message}
}

func NotFound(message string) error {
	return &AppError{StatusCode: http.StatusNotFound, Message: message}
}

func Conflict(message string) error {
	return &AppError{StatusCode: http.StatusConflict, Message: message}
}

func Internal(message string) error {
	return &AppError{StatusCode: http.StatusInternalServerError, Message: message}
}

func MethodNotAllowed() error {
	return &AppError{StatusCode: http.StatusMethodNotAllowed, Message: "method not allowed"}
}

func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

func GetStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

func As(err error, target *error) bool {
	return errors.As(err, target)
}
