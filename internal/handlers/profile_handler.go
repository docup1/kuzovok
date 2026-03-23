package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"kusovok/internal/application/profileapp"
	"kusovok/internal/domain/user"
	"kusovok/internal/infrastructure/auth"
	apperrors "kusovok/pkg/errors"
	"kusovok/pkg/response"
)

type ProfileHandler struct {
	getProfileUC           *profileapp.GetProfileUseCase
	getProfileByUsernameUC *profileapp.GetProfileByUsernameUseCase
	updateProfileUC        *profileapp.UpdateProfileUseCase
}

func NewProfileHandler(getProfileUC *profileapp.GetProfileUseCase, getProfileByUsernameUC *profileapp.GetProfileByUsernameUseCase, updateProfileUC *profileapp.UpdateProfileUseCase) *ProfileHandler {
	return &ProfileHandler{
		getProfileUC:           getProfileUC,
		getProfileByUsernameUC: getProfileByUsernameUC,
		updateProfileUC:        updateProfileUC,
	}
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromRequest(r)

	profile, err := h.getProfileUC.Execute(r.Context(), userID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", profile)
}

func (h *ProfileHandler) GetProfileByUsername(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	username := strings.TrimPrefix(path, "/api/users/")
	username = strings.Split(username, "?")[0]
	if username == "" || username == path {
		response.WriteError(w, http.StatusBadRequest, "username is required: "+path)
		return
	}

	profile, err := h.getProfileByUsernameUC.Execute(r.Context(), username)
	if err != nil {
		if apperrors.IsNoRows(err) {
			response.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", profile)
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Support _method override for proxy compatibility
	method := r.Method
	if method != http.MethodPut && method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.GetUserIDFromRequest(r)

	// Read body first to check for _method
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Check if _method is in body (for POST with method override)
	var methodCheck struct {
		Method string `json:"_method"`
	}
	if err := json.Unmarshal(body, &methodCheck); err == nil && methodCheck.Method != "" {
		method = methodCheck.Method
	}

	if method != http.MethodPut {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req user.UpdateProfileRequest
	if err := json.Unmarshal(body, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := h.updateProfileUC.Execute(r.Context(), userID, &req)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "Профиль обновлён", profile)
}
