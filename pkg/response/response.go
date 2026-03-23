package response

import (
	"encoding/json"
	"errors"
	"net/http"

	apperrors "kusovok/pkg/errors"
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
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}
	WriteError(w, http.StatusInternalServerError, "internal server error")
}

func BadRequest(message string) error {
	return apperrors.BadRequest(message)
}

func Unauthorized(message string) error {
	return apperrors.Unauthorized(message)
}

func Forbidden(message string) error {
	return apperrors.Forbidden(message)
}

func NotFound(message string) error {
	return apperrors.NotFound(message)
}

func Conflict(message string) error {
	return apperrors.Conflict(message)
}

func Internal(message string) error {
	return apperrors.Internal(message)
}
