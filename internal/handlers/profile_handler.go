package handlers

import (
	"encoding/json"
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
	if r.Method != http.MethodPut {
		response.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.GetUserIDFromRequest(r)

	var req user.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
