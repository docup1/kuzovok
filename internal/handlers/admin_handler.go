package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"kusovok/internal/application/admin"
	"kusovok/pkg/response"
)

type AdminHandler struct {
	getUsersUC *admin.GetUsersUseCase
	getLikesUC *admin.GetLikesUseCase
	manageUC   *admin.ManageAllowedUsersUseCase
	messages   AdminMessages
}

func NewAdminHandler(getUsersUC *admin.GetUsersUseCase, getLikesUC *admin.GetLikesUseCase, manageUC *admin.ManageAllowedUsersUseCase, messages AdminMessages) *AdminHandler {
	return &AdminHandler{
		getUsersUC: getUsersUC,
		getLikesUC: getLikesUC,
		manageUC:   manageUC,
		messages:   messages,
	}
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	users, err := h.getUsersUC.Execute(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", users)
}

func (h *AdminHandler) Likes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	likes, err := h.getLikesUC.Execute(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", likes)
}

func (h *AdminHandler) AllowedUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
		return
	}

	if req.UserID <= 0 {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidUserID)
		return
	}

	role := req.Role
	if strings.TrimSpace(role) == "" {
		role = "user"
	}

	user, err := h.manageUC.AddUser(r.Context(), req.UserID, role)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, h.messages.UserAdded, user)
}

func (h *AdminHandler) AllowedUserItem(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/admin/allowed-users/"

	pathValue := strings.TrimPrefix(r.URL.Path, prefix)
	pathValue = strings.Trim(pathValue, "/")
	if pathValue == "" {
		response.WriteError(w, http.StatusNotFound, h.messages.ErrorRouteNotFound)
		return
	}

	parts := strings.Split(pathValue, "/")
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || userID <= 0 {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidUserID)
		return
	}

	switch r.Method {
	case http.MethodPatch, http.MethodPost:
		if len(parts) == 2 && parts[1] == "role" {
			h.updateRole(w, r, userID)
		} else if len(parts) == 2 && parts[1] == "remove" {
			h.removeUser(w, r, userID)
		} else {
			response.WriteError(w, http.StatusNotFound, h.messages.ErrorRouteNotFound)
		}
	case http.MethodDelete:
		h.removeUser(w, r, userID)
	default:
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
	}
}

func (h *AdminHandler) updateRole(w http.ResponseWriter, r *http.Request, userID int64) {
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
		return
	}

	user, err := h.manageUC.UpdateRole(r.Context(), userID, req.Role)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, h.messages.RoleUpdated, user)
}

func (h *AdminHandler) removeUser(w http.ResponseWriter, r *http.Request, userID int64) {
	err := h.manageUC.RemoveUser(r.Context(), userID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, h.messages.UserRemoved, map[string]int64{"user_id": userID})
}
