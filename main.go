package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

const (
	jwtExpireHour        = 24
	imageLifetime        = 24 * time.Hour
	maxImageSize         = 10 << 20
	maxMultipartBodySize = maxImageSize + (1 << 20)
	cleanupInterval      = time.Minute
)

var (
	db                    *sql.DB
	jwtSecret             = getEnv("KUSOVOK_JWT_SECRET", "kusovok-secret-key-change-in-production")
	serverAddr            = getEnv("KUSOVOK_ADDR", ":8080")
	dbPath                = getEnv("KUSOVOK_DB_PATH", "./kusovok.db")
	cookiePath            = getEnv("KUSOVOK_COOKIE_PATH", "/")
	secureCookies         = strings.EqualFold(getEnv("KUSOVOK_SECURE_COOKIE", "false"), "true")
	imageDirPath          = getEnv("KUSOVOK_IMAGE_DIR", "./img")
	imagePublicPathPrefix = "/img/"
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type Post struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	Username       string     `json:"username"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	Likes          int        `json:"likes"`
	Liked          bool       `json:"liked"`
	ImageURL       *string    `json:"image_url"`
	ImageExpiresAt *time.Time `json:"image_expires_at"`
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

type postPayload struct {
	Content string
	Image   *imageUpload
}

type imageUpload struct {
	Bytes       []byte
	ContentType string
	Extension   string
}

type storedImage struct {
	URL       string
	ExpiresAt time.Time
}

type httpError struct {
	Status  int
	Message string
}

func (err *httpError) Error() string {
	return err.Message
}

func newHTTPError(status int, message string) error {
	return &httpError{Status: status, Message: message}
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
	if err := ensureImageDir(); err != nil {
		log.Fatal(err)
	}
	if err := cleanupExpiredImages(time.Now().UTC()); err != nil {
		log.Printf("Image cleanup error: %v", err)
	}
	startExpiredImageCleaner()

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

	if err := ensurePostImageColumns(); err != nil {
		return err
	}
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_image_expires_at ON posts(image_expires_at)")
	return err
}

func ensurePostImageColumns() error {
	rows, err := db.Query("PRAGMA table_info(posts)")
	if err != nil {
		return err
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !columns["image_url"] {
		if _, err := db.Exec("ALTER TABLE posts ADD COLUMN image_url TEXT"); err != nil {
			return err
		}
	}
	if !columns["image_expires_at"] {
		if _, err := db.Exec("ALTER TABLE posts ADD COLUMN image_expires_at DATETIME"); err != nil {
			return err
		}
	}
	return nil
}

func ensureImageDir() error {
	return os.MkdirAll(imageDirPath, 0o755)
}

func startExpiredImageCleaner() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := cleanupExpiredImages(time.Now().UTC()); err != nil {
				log.Printf("Image cleanup error: %v", err)
			}
		}
	}()
}

func cleanupExpiredImages(now time.Time) error {
	rows, err := db.Query(
		"SELECT id, image_url FROM posts WHERE image_url IS NOT NULL AND image_url != '' AND image_expires_at IS NOT NULL AND image_expires_at <= ?",
		now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredImage struct {
		postID   int64
		imageURL string
	}

	expiredImages := []expiredImage{}
	var firstErr error
	for rows.Next() {
		var item expiredImage
		if err := rows.Scan(&item.postID, &item.imageURL); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		expiredImages = append(expiredImages, item)
	}

	if err := rows.Err(); err != nil && firstErr == nil {
		firstErr = err
	}

	for _, item := range expiredImages {
		if err := deleteImageByURL(item.imageURL); err != nil {
			log.Printf("Image delete error for post %d: %v", item.postID, err)
			if firstErr == nil {
				firstErr = err
			}
		}

		if _, err := db.Exec("UPDATE posts SET image_url = NULL, image_expires_at = NULL WHERE id = ?", item.postID); err != nil {
			log.Printf("Image cleanup db error for post %d: %v", item.postID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func newServerMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/img/", imageHandler)
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

func imageHandler(w http.ResponseWriter, r *http.Request) {
	filePath, err := resolveImageFilePath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	http.ServeFile(w, r, filePath)
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
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{Success: false, Message: message})
}

func writeSuccess(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: message, Data: data})
}

func writeHandlerError(w http.ResponseWriter, err error) {
	var clientErr *httpError
	if errors.As(err, &clientErr) {
		writeError(w, clientErr.Status, clientErr.Message)
		return
	}
	log.Printf("Unexpected handler error: %v", err)
	writeError(w, http.StatusInternalServerError, "Ошибка сервера")
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
	_ = db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&postCount)
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

	payload, err := parseCreatePostRequest(w, r)
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	content := strings.TrimSpace(payload.Content)
	if content == "" && payload.Image == nil {
		writeError(w, http.StatusBadRequest, "Пост должен содержать текст или картинку")
		return
	}
	if len(content) > 1000 {
		writeError(w, http.StatusBadRequest, "Пост слишком длинный (макс. 1000 символов)")
		return
	}

	createdAt := time.Now().UTC()
	var image *storedImage
	if payload.Image != nil {
		image, err = storeImage(payload.Image, createdAt)
		if err != nil {
			writeHandlerError(w, err)
			return
		}
	}

	var imageURL interface{}
	var imageExpiresAt interface{}
	if image != nil {
		imageURL = image.URL
		imageExpiresAt = image.ExpiresAt.Format(time.RFC3339)
	}

	result, err := db.Exec(
		"INSERT INTO posts (user_id, content, created_at, image_url, image_expires_at) VALUES (?, ?, ?, ?, ?)",
		userID,
		content,
		createdAt.Format(time.RFC3339),
		imageURL,
		imageExpiresAt,
	)
	if err != nil {
		if image != nil {
			_ = deleteImageByURL(image.URL)
		}
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
		return
	}

	postID, _ := result.LastInsertId()
	post := Post{
		ID:        postID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: createdAt,
		Likes:     0,
	}
	if image != nil {
		post.ImageURL = &image.URL
		post.ImageExpiresAt = &image.ExpiresAt
	}

	writeSuccess(w, "Пост создан", post)
}

func parseCreatePostRequest(w http.ResponseWriter, r *http.Request) (postPayload, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartPostRequest(w, r)
	}
	return parseJSONPostRequest(r)
}

func parseJSONPostRequest(r *http.Request) (postPayload, error) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return postPayload{}, newHTTPError(http.StatusBadRequest, "Неверный формат данных")
	}
	return postPayload{Content: req.Content}, nil
}

func parseMultipartPostRequest(w http.ResponseWriter, r *http.Request) (postPayload, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartBodySize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		if isRequestTooLargeError(err) {
			return postPayload{}, newHTTPError(http.StatusBadRequest, "Картинка слишком большая (макс. 10 MB)")
		}
		return postPayload{}, newHTTPError(http.StatusBadRequest, "Неверный формат формы")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	payload := postPayload{Content: r.FormValue("content")}
	file, _, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return payload, nil
		}
		return postPayload{}, newHTTPError(http.StatusBadRequest, "Не удалось прочитать картинку")
	}
	defer file.Close()

	image, err := readImageUpload(file)
	if err != nil {
		return postPayload{}, err
	}
	payload.Image = image
	return payload, nil
}

func isRequestTooLargeError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr) || strings.Contains(strings.ToLower(err.Error()), "request body too large")
}

func readImageUpload(reader io.Reader) (*imageUpload, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxImageSize+1))
	if err != nil {
		return nil, newHTTPError(http.StatusBadRequest, "Не удалось прочитать картинку")
	}
	if len(data) == 0 {
		return nil, newHTTPError(http.StatusBadRequest, "Файл картинки пустой")
	}
	if int64(len(data)) > maxImageSize {
		return nil, newHTTPError(http.StatusBadRequest, "Картинка слишком большая (макс. 10 MB)")
	}

	contentType := http.DetectContentType(data)
	extension, ok := allowedImageTypes[contentType]
	if !ok {
		return nil, newHTTPError(http.StatusBadRequest, "Допустимы только JPG, PNG, WEBP или GIF")
	}

	return &imageUpload{
		Bytes:       data,
		ContentType: contentType,
		Extension:   extension,
	}, nil
}

func storeImage(image *imageUpload, createdAt time.Time) (*storedImage, error) {
	if err := ensureImageDir(); err != nil {
		return nil, err
	}

	fileName := uuid.NewString() + image.Extension
	filePath := filepath.Join(imageDirPath, fileName)
	if err := os.WriteFile(filePath, image.Bytes, 0o644); err != nil {
		return nil, err
	}

	return &storedImage{
		URL:       path.Join(imagePublicPathPrefix, fileName),
		ExpiresAt: createdAt.Add(imageLifetime),
	}, nil
}

func resolveImageFilePath(imageURL string) (string, error) {
	fileName := strings.TrimPrefix(imageURL, imagePublicPathPrefix)
	if fileName == imageURL || fileName == "" {
		return "", fmt.Errorf("invalid image url: %s", imageURL)
	}
	if fileName != path.Base(fileName) {
		return "", fmt.Errorf("invalid image file name: %s", fileName)
	}

	cleanDir := filepath.Clean(imageDirPath)
	filePath := filepath.Clean(filepath.Join(cleanDir, fileName))
	prefix := cleanDir + string(os.PathSeparator)
	if filePath != cleanDir && !strings.HasPrefix(filePath, prefix) {
		return "", fmt.Errorf("invalid image path: %s", imageURL)
	}
	return filePath, nil
}

func deleteImageByURL(imageURL string) error {
	filePath, err := resolveImageFilePath(imageURL)
	if err != nil {
		return err
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	posts, err := queryPosts(
		"SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes, EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked, p.image_url, p.image_expires_at FROM posts p JOIN users u ON p.user_id = u.id WHERE p.user_id = ? ORDER BY p.created_at DESC",
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
		"SELECT p.id, p.user_id, u.username, p.content, p.created_at, (SELECT COUNT(*) FROM likes WHERE post_id = p.id) AS likes, EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked, p.image_url, p.image_expires_at FROM posts p JOIN users u ON p.user_id = u.id ORDER BY p.created_at DESC LIMIT 50",
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
		var imageURL sql.NullString
		var imageExpiresAt sql.NullString
		if err := rows.Scan(&post.ID, &post.UserID, &post.Username, &post.Content, &createdAt, &likes, &liked, &imageURL, &imageExpiresAt); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		post.CreatedAt = parseTimestamp(createdAt)
		post.Likes = likes
		post.Liked = liked == 1
		post.ImageURL = nullableStringPointer(imageURL)
		post.ImageExpiresAt = nullableTimePointer(imageExpiresAt)
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := value.String
	return &result
}

func nullableTimePointer(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed := parseTimestamp(value.String)
	return &parsed
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
