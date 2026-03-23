package handlers

import (
	"encoding/json"
	"net/http"

	"kusovok/internal/application/authapp"
	"kusovok/internal/infrastructure/auth"
	"kusovok/pkg/response"
)

type UserHandler struct {
	registerUC *appauth.RegisterUseCase
	loginUC    *appauth.LoginUseCase
	meUC       *appauth.MeUseCase
	cookie     *auth.CookieService
	messages   UserMessages
}

func NewUserHandler(registerUC *appauth.RegisterUseCase, loginUC *appauth.LoginUseCase, meUC *appauth.MeUseCase, cookie *auth.CookieService, messages UserMessages) *UserHandler {
	return &UserHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		meUC:       meUC,
		cookie:     cookie,
		messages:   messages,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
		return
	}

	token, authResp, err := h.registerUC.Execute(r.Context(), req.Username, req.Password)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	h.cookie.SetToken(w, token)
	response.WriteSuccess(w, h.messages.RegisterSuccess, authResp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
		return
	}

	token, authResp, err := h.loginUC.Execute(r.Context(), req.Username, req.Password)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	h.cookie.SetToken(w, token)
	response.WriteSuccess(w, h.messages.LoginSuccess, authResp)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.cookie.Clear(w)
	response.WriteSuccess(w, h.messages.LogoutSuccess, nil)
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromRequest(r)
	username := r.Header.Get("X-Username")

	me, err := h.meUC.Execute(r.Context(), userID, username)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", me)
}
