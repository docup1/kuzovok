package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

const (
	jwtSecret     = "kusovok-secret-key-change-in-production"
	jwtExpireHour = 24
)

var db *sql.DB

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
	db, err = sql.Open("sqlite", "./kusovok.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDB()
	http.HandleFunc("/", staticHandler)
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/logout", logoutHandler)
	http.HandleFunc("/api/me", authMiddleware(meHandler))
	http.HandleFunc("/api/posts", authMiddleware(postsHandler))
	http.HandleFunc("/api/feed", authMiddleware(feedHandler))
	http.HandleFunc("/api/like", authMiddleware(likeHandler))
	fmt.Println("🐠 Кузовок запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)")
	db.Exec("CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id))")
	db.Exec("CREATE TABLE IF NOT EXISTS likes (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, post_id INTEGER NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id), FOREIGN KEY (post_id) REFERENCES posts(id), UNIQUE(user_id, post_id))")
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
	http.SetCookie(w, &http.Cookie{Name: "token", Value: token, Path: "/", MaxAge: jwtExpireHour * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode})
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
	http.SetCookie(w, &http.Cookie{Name: "token", Value: token, Path: "/", MaxAge: jwtExpireHour * 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	writeSuccess(w, "Вход успешен", map[string]interface{}{"id": user.ID, "username": user.Username})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
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
	var req struct{ Content string }
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
	rows, err := db.Query("SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes FROM posts p JOIN users u ON p.user_id = u.id WHERE p.user_id = ? ORDER BY p.created_at DESC", userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer rows.Close()
	posts := []Post{}
	for rows.Next() {
		var post Post
		var createdAt string
		var likes int
		if err := rows.Scan(&post.ID, &post.UserID, &post.Username, &post.Content, &createdAt, &likes); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		post.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			post.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err != nil {
			log.Printf("Parse error: %v, createdAt: %s", err, createdAt)
			post.CreatedAt = time.Now()
		}
		post.Likes = likes
		posts = append(posts, post)
	}
	writeSuccess(w, "", posts)
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) as likes FROM posts p JOIN users u ON p.user_id = u.id ORDER BY p.created_at DESC LIMIT 50")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}
	defer rows.Close()
	posts := []Post{}
	for rows.Next() {
		var post Post
		var createdAt string
		var likes int
		if err := rows.Scan(&post.ID, &post.UserID, &post.Username, &post.Content, &createdAt, &likes); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		post.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			post.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err != nil {
			log.Printf("Parse error: %v, createdAt: %s", err, createdAt)
			post.CreatedAt = time.Now()
		}
		post.Likes = likes
		posts = append(posts, post)
	}
	writeSuccess(w, "", posts)
}

func likeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Метод не разрешен")
		return
	}
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	log.Printf("Like request: userID=%d", userID)
	var req struct{ PostID int64 }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Decode error: %v", err)
		writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}
	log.Printf("PostID=%d", req.PostID)
	var postExists int
	db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", req.PostID).Scan(&postExists)
	if postExists == 0 {
		writeError(w, http.StatusNotFound, "Пост не найден")
		return
	}
	_, err := db.Exec("INSERT INTO likes (user_id, post_id) VALUES (?, ?)", userID, req.PostID)
	if err != nil {
		log.Printf("Insert like error: %v", err)
		db.Exec("DELETE FROM likes WHERE user_id = ? AND post_id = ?", userID, req.PostID)
	}
	var likes int
	db.QueryRow("SELECT COUNT(*) FROM likes WHERE post_id = ?", req.PostID).Scan(&likes)
	log.Printf("Total likes=%d", likes)
	writeSuccess(w, "", map[string]interface{}{"likes": likes})
}
