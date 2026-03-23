package handlers

import (
	"context"
	"net/http"
	"strconv"

	"kusovok/internal/domain/access"
	"kusovok/internal/domain/shared"
	"kusovok/internal/infrastructure/auth"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
	cookieName string
	accessSvc  AccessService
	messages   AuthMessages
}

type AccessService interface {
	GetAccessInfo(ctx context.Context, userID int64) (access.AccessInfo, error)
}

func NewAuthMiddleware(jwtService *auth.JWTService, cookieName string, accessSvc AccessService, messages AuthMessages) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		cookieName: cookieName,
		accessSvc:  accessSvc,
		messages:   messages,
	}
}

func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, m.messages.ErrorUnauthorized)
			return
		}

		claims, err := m.jwtService.ValidateToken(cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, m.messages.ErrorInvalidToken)
			return
		}

		r.Header.Set("X-User-ID", strconv.FormatInt(claims.UserID, 10))
		r.Header.Set("X-Username", claims.Username)
		next(w, r)
	}
}

func (m *AuthMiddleware) RequireAllowedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		if err != nil || userID <= 0 {
			writeError(w, http.StatusUnauthorized, m.messages.ErrorUnauthorized)
			return
		}

		accessInfo, err := m.accessSvc.GetAccessInfo(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, m.messages.ErrorUnauthorized)
			return
		}

		if !accessInfo.IsAllowed {
			writeError(w, http.StatusForbidden, m.messages.AccessDenied)
			return
		}

		next(w, r)
	}
}

func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		if err != nil || userID <= 0 {
			writeError(w, http.StatusUnauthorized, m.messages.ErrorUnauthorized)
			return
		}

		accessInfo, err := m.accessSvc.GetAccessInfo(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, m.messages.ErrorUnauthorized)
			return
		}

		if !accessInfo.IsAllowed || accessInfo.Role != shared.RoleAdmin {
			writeError(w, http.StatusForbidden, m.messages.AdminDenied)
			return
		}

		next(w, r)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"message":"` + message + `"}`))
}
