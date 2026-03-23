package handlers

import (
	"encoding/json"
	"net/http"

	"kusovok/internal/application/likeapp"
	"kusovok/internal/infrastructure/auth"
	"kusovok/pkg/response"
)

type LikeHandler struct {
	toggleUC *likeapp.ToggleLikeUseCase
	messages LikeMessages
}

func NewLikeHandler(toggleUC *likeapp.ToggleLikeUseCase, messages LikeMessages) *LikeHandler {
	return &LikeHandler{
		toggleUC: toggleUC,
		messages: messages,
	}
}

func (h *LikeHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	var req struct {
		PostID int64 `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
		return
	}

	if req.PostID <= 0 {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidPostID)
		return
	}

	userID := auth.GetUserIDFromRequest(r)

	result, err := h.toggleUC.Execute(r.Context(), userID, req.PostID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", result)
}
