package auth

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

type CookieService struct {
	name   string
	path   string
	secure bool
	maxAge int
}

func NewCookieService(name, path string, secure bool, maxAgeHours int) *CookieService {
	return &CookieService{
		name:   name,
		path:   path,
		secure: secure,
		maxAge: maxAgeHours * 3600,
	}
}

func (s *CookieService) Name() string {
	return s.name
}

func (s *CookieService) SetToken(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.name,
		Value:    token,
		Path:     s.path,
		MaxAge:   s.maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func (s *CookieService) Clear(w http.ResponseWriter) {
	s.SetToken(w, "")
}

func (s *CookieService) GetToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(s.name)
	if err != nil {
		return "", errors.New("cookie not found")
	}
	return cookie.Value, nil
}

func GetUserID(r *http.Request) (int64, error) {
	idStr := r.Header.Get("X-User-ID")
	if idStr == "" {
		return 0, errors.New("user id not found")
	}
	return strconv.ParseInt(idStr, 10, 64)
}

func GetUserIDFromRequest(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	return id
}

func GetUsername(r *http.Request) string {
	return r.Header.Get("X-Username")
}

func SetUserHeaders(r *http.Request, userID int64, username string) {
	r.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
	r.Header.Set("X-Username", username)
}

func ParseTimestamp(value string) time.Time {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Now()
}
