package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

const (
	jwtExpireHour = 24
)

var (
	db            *sql.DB
	jwtSecret     = getEnv("KUSOVOK_JWT_SECRET", "kusovok-secret-key-change-in-production")
	serverAddr    = getEnv("KUSOVOK_ADDR", ":8080")
	dbPath        = getEnv("KUSOVOK_DB_PATH", "./kusovok.db")
	cookiePath    = getEnv("KUSOVOK_COOKIE_PATH", "/")
	secureCookies = strings.EqualFold(getEnv("KUSOVOK_SECURE_COOKIE", "false"), "true")
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type Post struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Likes     int       `json:"likes"`
	Liked     bool      `json:"liked"`
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	var err error
	db, err = openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := initDB(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("🐠 Кузовок запущен на %s\n", publicServerURL(serverAddr))
	log.Fatal(http.ListenAndServe(serverAddr, newServerMux()))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func openDB(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	return database, nil
}

func initDB() error {
	statements := []string{
		"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id))",
		"CREATE TABLE IF NOT EXISTS likes (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, post_id INTEGER NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id), FOREIGN KEY (post_id) REFERENCES posts(id), UNIQUE(user_id, post_id))",
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func newServerMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", staticHandler)
	mux.HandleFunc("/api/register", registerHandler)
	mux.HandleFunc("/api/login", loginHandler)
	mux.HandleFunc("/api/logout", logoutHandler)
	mux.HandleFunc("/api/me", authMiddleware(meHandler))
	mux.HandleFunc("/api/posts", authMiddleware(postsHandler))
	mux.HandleFunc("/api/feed", authMiddleware(feedHandler))
	mux.HandleFunc("/api/like", authMiddleware(likeHandler))
	return mux
}

func setAuthCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     cookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies,
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	setAuthCookie(w, "", -1)
}

func publicServerURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		http.ServeFile(w, r, "static/index.html")
		return
	}
	http.ServeFile(w, r, "static"+r.URL.Path)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{Success: false, Message: message})
}

func writeSuccess(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: message, Data: data})
}

func generateToken(userID int64, username string) (string, error) {
	claims := Claims{UserID: userID, Username: username, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * jwtExpireHour)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func getUserFromToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Требуется авторизация")
			return
		}
		claims, err := getUserFromToken(cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Неверный токен")
			return
		}
		r.Header.Set("X-User-ID", strconv.FormatInt(claims.UserID, 10))
		r.Header.Set("X-Username", claims.Username)
		next(w, r)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Метод не разрешен")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Все поля обязательны")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "Пароль должен быть не менее 6 символов")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	result, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", req.Username, string(hashedPassword))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeError(w, http.StatusConflict, "Пользователь уже существует")
			return
		}
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	userID, _ := result.LastInsertId()
	token, _ := generateToken(userID, req.Username)
	setAuthCookie(w, token, jwtExpireHour*3600)
	writeSuccess(w, "Регистрация успешна", map[string]interface{}{"id": userID, "username": req.Username})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Метод не разрешен")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}
	var user User
	err := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", req.Username).Scan(&user.ID, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	token, err := generateToken(user.ID, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	setAuthCookie(w, token, jwtExpireHour*3600)
	writeSuccess(w, "Вход успешен", map[string]interface{}{"id": user.ID, "username": user.Username})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)
	writeSuccess(w, "Выход успешен", nil)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	username := r.Header.Get("X-Username")
	var postCount int
	db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&postCount)
	writeSuccess(w, "", map[string]interface{}{"id": userID, "username": username, "post_count": postCount})
}

func postsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createPost(w, r)
	case http.MethodGet:
		getUserPosts(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Метод не разрешен")
	}
}

func createPost(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	username := r.Header.Get("X-Username")
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeError(w, http.StatusBadRequest, "Пост не может быть пустым")
		return
	}
	if len(content) > 1000 {
		writeError(w, http.StatusBadRequest, "Пост слишком длинный (макс. 1000 символов)")
		return
	}
	result, err := db.Exec("INSERT INTO posts (user_id, content) VALUES (?, ?)", userID, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	postID, _ := result.LastInsertId()
	post := Post{ID: postID, UserID: userID, Username: username, Content: content, CreatedAt: time.Now(), Likes: 0}
	writeSuccess(w, "Пост создан", post)
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	posts, err := queryPosts(
		"SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes, EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked FROM posts p JOIN users u ON p.user_id = u.id WHERE p.user_id = ? ORDER BY p.created_at DESC",
		userID,
		userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	writeSuccess(w, "", posts)
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	posts, err := queryPosts(
		"SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes, EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked FROM posts p JOIN users u ON p.user_id = u.id ORDER BY p.created_at DESC LIMIT 50",
		currentUserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	writeSuccess(w, "", posts)
}

func likeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Метод не разрешен")
		return
	}
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	var req struct {
		PostID int64 `json:"post_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}
	if req.PostID <= 0 {
		writeError(w, http.StatusBadRequest, "Неверный идентификатор поста")
		return
	}
	var postExists int
	if err := db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", req.PostID).Scan(&postExists); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	if postExists == 0 {
		writeError(w, http.StatusNotFound, "Пост не найден")
		return
	}
	liked := true
	_, err := db.Exec("INSERT INTO likes (user_id, post_id) VALUES (?, ?)", userID, req.PostID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			liked = false
			if _, deleteErr := db.Exec("DELETE FROM likes WHERE user_id = ? AND post_id = ?", userID, req.PostID); deleteErr != nil {
				log.Printf("Delete like error: %v", deleteErr)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
		} else {
			log.Printf("Insert like error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
	}
	var likes int
	if err := db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", req.PostID).Scan(&likes); err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	writeSuccess(w, "", map[string]interface{}{"likes": likes, "liked": liked})
}

func queryPosts(query string, args ...interface{}) ([]Post, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var post Post
		var createdAt string
		var likes int
		var liked int
		if err := rows.Scan(&post.ID, &post.UserID, &post.Username, &post.Content, &createdAt, &likes, &liked); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		post.CreatedAt = parseTimestamp(createdAt)
		post.Likes = likes
		post.Liked = liked == 1
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func parseTimestamp(value string) time.Time {
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
	log.Printf("Parse error: unsupported timestamp format %q", value)
	return time.Now()
}
