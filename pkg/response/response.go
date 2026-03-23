package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteSuccess(w http.ResponseWriter, message string, data interface{}) {
	WriteJSON(w, http.StatusOK, Response{Success: true, Message: message, Data: data})
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, Response{Success: false, Message: message})
}

func WriteAppError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if As(err, &appErr) {
		WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}
	WriteError(w, http.StatusInternalServerError, "internal server error")
}

type AppError struct {
	StatusCode int
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(statusCode int, message string) error {
	return &AppError{StatusCode: statusCode, Message: message}
}

func BadRequest(message string) error {
	return New(http.StatusBadRequest, message)
}

func Unauthorized(message string) error {
	return New(http.StatusUnauthorized, message)
}

func Forbidden(message string) error {
	return New(http.StatusForbidden, message)
}

func NotFound(message string) error {
	return New(http.StatusNotFound, message)
}

func Conflict(message string) error {
	return New(http.StatusConflict, message)
}

func Internal(message string) error {
	return New(http.StatusInternalServerError, message)
}

func As(err error, target interface{}) bool {
	switch e := err.(type) {
	case *AppError:
		if t, ok := target.(**AppError); ok {
			*t = e
			return true
		}
	}
	return false
}
