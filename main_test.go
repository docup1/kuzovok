package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type likeResponse struct {
	Likes int  `json:"likes"`
	Liked bool `json:"liked"`
}

func TestInitDBMigratesExistingPostsSchema(t *testing.T) {
	rootDir := t.TempDir()
	legacyDB := openTestDB(t, filepath.Join(rootDir, "legacy.db"))
	defer legacyDB.Close()

	previousDB := db
	db = legacyDB
	t.Cleanup(func() {
		db = previousDB
	})

	legacyStatements := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE posts (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id))",
		"CREATE TABLE likes (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, post_id INTEGER NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id), FOREIGN KEY (post_id) REFERENCES posts(id), UNIQUE(user_id, post_id))",
		"CREATE TABLE allowed_users (user_id INTEGER PRIMARY KEY, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE)",
	}
	for _, statement := range legacyStatements {
		if _, err := legacyDB.Exec(statement); err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}
	if _, err := legacyDB.Exec("INSERT INTO users (username, password) VALUES ('legacy-admin', 'hash')"); err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	if _, err := legacyDB.Exec("INSERT INTO allowed_users (user_id) VALUES (1)"); err != nil {
		t.Fatalf("insert legacy allowlist: %v", err)
	}

	if err := initDB(); err != nil {
		t.Fatalf("init db migration: %v", err)
	}

	columns := tableColumns(t, legacyDB, "posts")
	if !columns["image_url"] || !columns["image_expires_at"] {
		t.Fatalf("expected migrated posts columns, got %+v", columns)
	}
	allowedColumns := tableColumns(t, legacyDB, "allowed_users")
	if !allowedColumns["user_id"] || !allowedColumns["created_at"] || !allowedColumns["role"] {
		t.Fatalf("expected allowed_users table to be created, got %+v", allowedColumns)
	}
	var legacyRole string
	if err := legacyDB.QueryRow("SELECT role FROM allowed_users WHERE user_id = 1").Scan(&legacyRole); err != nil {
		t.Fatalf("query migrated allowlist role: %v", err)
	}
	if legacyRole != string(roleUser) {
		t.Fatalf("expected migrated allowlist role to default to %q, got %q", roleUser, legacyRole)
	}

	if !hasIndex(t, legacyDB, "posts", "idx_posts_image_expires_at") {
		t.Fatalf("expected idx_posts_image_expires_at to be created")
	}
}

func TestRegisterCreatePostAndToggleLike(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	alice := newSessionClient(t)
	registerUser(t, alice, server.URL, "alice", "secret12")
	allowUserByUsername(t, "alice")
	aliceMe := fetchMe(t, alice, server.URL)
	if !aliceMe.IsAllowed || aliceMe.Role != string(roleUser) {
		t.Fatalf("expected alice to be allowed after allowlist insert")
	}
	post := createPostForTest(t, alice, server.URL, "Первый локальный пост")

	bob := newSessionClient(t)
	registerUser(t, bob, server.URL, "bob", "secret12")
	allowUserByUsername(t, "bob")
	bobMe := fetchMe(t, bob, server.URL)
	if !bobMe.IsAllowed || bobMe.Role != string(roleUser) {
		t.Fatalf("expected bob to be allowed after allowlist insert")
	}

	firstLike := likePostForTest(t, bob, server.URL, post.ID)
	if !firstLike.Liked || firstLike.Likes != 1 {
		t.Fatalf("expected first like to set liked=true and likes=1, got %+v", firstLike)
	}

	feedAfterLike := fetchFeed(t, bob, server.URL)
	if len(feedAfterLike) != 1 {
		t.Fatalf("expected 1 post in feed after like, got %d", len(feedAfterLike))
	}
	if !feedAfterLike[0].Liked || feedAfterLike[0].Likes != 1 {
		t.Fatalf("expected liked post in feed, got %+v", feedAfterLike[0])
	}

	secondLike := likePostForTest(t, bob, server.URL, post.ID)
	if secondLike.Liked || secondLike.Likes != 0 {
		t.Fatalf("expected second like to remove like and reset likes to 0, got %+v", secondLike)
	}

	feedAfterUnlike := fetchFeed(t, bob, server.URL)
	if len(feedAfterUnlike) != 1 {
		t.Fatalf("expected 1 post in feed after unlike, got %d", len(feedAfterUnlike))
	}
	if feedAfterUnlike[0].Liked || feedAfterUnlike[0].Likes != 0 {
		t.Fatalf("expected unliked post in feed, got %+v", feedAfterUnlike[0])
	}
}

func TestCreatePostWithImageAndCleanup(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := newSessionClient(t)
	registerUser(t, client, server.URL, "nemo", "secret12")
	allowUserByUsername(t, "nemo")

	textOnly := createPostForTest(t, client, server.URL, "Только текст")
	if textOnly.ImageURL != nil || textOnly.ImageExpiresAt != nil {
		t.Fatalf("expected text-only post to have no image metadata, got %+v", textOnly)
	}

	imagePost := createMultipartPostForTest(t, client, server.URL, "Пост с картинкой", "reef.png", pngFixture())
	if imagePost.ImageURL == nil || imagePost.ImageExpiresAt == nil {
		t.Fatalf("expected post image metadata, got %+v", imagePost)
	}

	imageOnlyPost := createMultipartPostForTest(t, client, server.URL, "", "coral.png", pngFixture())
	if imageOnlyPost.Content != "" {
		t.Fatalf("expected image-only post to keep empty content, got %q", imageOnlyPost.Content)
	}
	if imageOnlyPost.ImageURL == nil || imageOnlyPost.ImageExpiresAt == nil {
		t.Fatalf("expected image-only post image metadata, got %+v", imageOnlyPost)
	}

	imageBody, status := getBinary(t, client, server.URL+*imagePost.ImageURL)
	if status != http.StatusOK {
		t.Fatalf("expected uploaded image to be served, got status %d", status)
	}
	if !bytes.Equal(imageBody, pngFixture()) {
		t.Fatalf("unexpected image body returned by server")
	}

	localPath, err := resolveImageFilePath(*imageOnlyPost.ImageURL)
	if err != nil {
		t.Fatalf("resolve image path: %v", err)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("expected uploaded image file to exist: %v", err)
	}

	feed := fetchFeed(t, client, server.URL)
	if findPostByID(feed, imagePost.ID).ImageURL == nil {
		t.Fatalf("expected feed to include image metadata")
	}
	if findPostByID(feed, imageOnlyPost.ID).ImageURL == nil {
		t.Fatalf("expected feed to include image-only metadata")
	}

	expiredAt := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	if _, err := db.Exec("UPDATE posts SET image_expires_at = ? WHERE id = ?", expiredAt, imageOnlyPost.ID); err != nil {
		t.Fatalf("expire image in db: %v", err)
	}

	if err := cleanupExpiredImages(time.Now().UTC()); err != nil {
		t.Fatalf("cleanup expired images: %v", err)
	}
	if err := cleanupExpiredImages(time.Now().UTC()); err != nil {
		t.Fatalf("cleanup should be idempotent: %v", err)
	}

	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatalf("expected expired image file to be deleted, got %v", err)
	}

	_, expiredStatus := getBinary(t, client, server.URL+*imageOnlyPost.ImageURL)
	if expiredStatus != http.StatusNotFound {
		t.Fatalf("expected deleted image to return 404, got %d", expiredStatus)
	}

	feedAfterCleanup := fetchFeed(t, client, server.URL)
	expiredPost := findPostByID(feedAfterCleanup, imageOnlyPost.ID)
	if expiredPost.ImageURL != nil || expiredPost.ImageExpiresAt != nil {
		t.Fatalf("expected expired image metadata to be cleared, got %+v", expiredPost)
	}

	activePost := findPostByID(feedAfterCleanup, imagePost.ID)
	if activePost.ImageURL == nil || activePost.ImageExpiresAt == nil {
		t.Fatalf("expected active image metadata to remain, got %+v", activePost)
	}
}

func TestCreatePostRejectsInvalidImage(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := newSessionClient(t)
	registerUser(t, client, server.URL, "dory", "secret12")
	allowUserByUsername(t, "dory")

	response := createMultipartPostExpectFailure(t, client, server.URL, "Невалидная картинка", "bad.txt", []byte("not an image"))
	if !strings.Contains(response.Message, "Допустимы только") {
		t.Fatalf("expected invalid mime error, got %q", response.Message)
	}

	assertImageDirEmpty(t)
}

func TestCreatePostRejectsOversizedImage(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := newSessionClient(t)
	registerUser(t, client, server.URL, "marlin", "secret12")
	allowUserByUsername(t, "marlin")

	largeImage := bytes.Repeat([]byte("a"), maxImageSize+1024)
	response := createMultipartPostExpectFailure(t, client, server.URL, "Слишком большая картинка", "huge.png", largeImage)
	if !strings.Contains(response.Message, "слишком большая") {
		t.Fatalf("expected oversize error, got %q", response.Message)
	}

	assertImageDirEmpty(t)
}

func TestUnallowedUserCannotUseProtectedAPI(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	allowedAuthor := newSessionClient(t)
	registerUser(t, allowedAuthor, server.URL, "allowed", "secret12")
	allowUserByUsername(t, "allowed")
	post := createPostForTest(t, allowedAuthor, server.URL, "Пост для проверки доступа")

	blocked := newSessionClient(t)
	registerUser(t, blocked, server.URL, "blocked", "secret12")

	me := fetchMe(t, blocked, server.URL)
	if me.IsAllowed {
		t.Fatalf("expected blocked user to stay disallowed, got %+v", me)
	}
	if me.Role != "" {
		t.Fatalf("expected blocked user to have empty role, got %q", me.Role)
	}
	if me.AccessMessage != accessDeniedMessage {
		t.Fatalf("expected access denied message %q, got %q", accessDeniedMessage, me.AccessMessage)
	}

	assertForbiddenJSON(t, blocked, http.MethodGet, server.URL+"/api/feed", nil)
	assertForbiddenJSON(t, blocked, http.MethodGet, server.URL+"/api/posts", nil)
	assertForbiddenJSON(t, blocked, http.MethodPost, server.URL+"/api/posts", map[string]string{"content": "Не должен сохраниться"})
	assertForbiddenJSON(t, blocked, http.MethodPost, server.URL+"/api/like", map[string]int64{"post_id": post.ID})
}

func TestAdminEndpointsRequireAdminRole(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	guest := newSessionClient(t)
	assertJSONStatus(t, guest, http.MethodGet, server.URL+"/api/admin/users", nil, http.StatusUnauthorized)

	allowedUser := newSessionClient(t)
	registerUser(t, allowedUser, server.URL, "regular", "secret12")
	allowUserByUsername(t, "regular")
	me := fetchMe(t, allowedUser, server.URL)
	if me.Role != string(roleUser) {
		t.Fatalf("expected regular user role %q, got %q", roleUser, me.Role)
	}

	var response apiResponse
	status := requestJSONStatus(t, allowedUser, http.MethodGet, server.URL+"/api/admin/users", nil, &response)
	if status != http.StatusForbidden {
		t.Fatalf("expected admin endpoint to return 403 for non-admin, got %d", status)
	}
	if response.Message != adminDeniedMessage {
		t.Fatalf("expected admin denied message %q, got %q", adminDeniedMessage, response.Message)
	}
}

func TestAdminCanManageUsersAndLikes(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	adminClient := newSessionClient(t)
	registerUser(t, adminClient, server.URL, "captain", "secret12")
	allowUserByUsernameAs(t, "captain", roleAdmin)

	captainMe := fetchMe(t, adminClient, server.URL)
	if captainMe.Role != string(roleAdmin) {
		t.Fatalf("expected captain role %q, got %q", roleAdmin, captainMe.Role)
	}

	author := newSessionClient(t)
	registerUser(t, author, server.URL, "author", "secret12")
	allowUserByUsername(t, "author")
	post := createPostForTest(t, author, server.URL, "Пост для админской аналитики")

	liker := newSessionClient(t)
	registerUser(t, liker, server.URL, "liker", "secret12")
	allowUserByUsername(t, "liker")
	likePostForTest(t, liker, server.URL, post.ID)

	candidate := newSessionClient(t)
	registerUser(t, candidate, server.URL, "candidate", "secret12")
	candidateID := userIDByUsername(t, "candidate")

	initialUsers := fetchAdminUsers(t, adminClient, server.URL)
	if findAdminUser(initialUsers, userIDByUsername(t, "captain")).Role != string(roleAdmin) {
		t.Fatalf("expected admin list to include captain as admin")
	}

	addedUser := addAllowedUserForTest(t, adminClient, server.URL, candidateID)
	if !addedUser.IsAllowed || addedUser.Role != string(roleUser) {
		t.Fatalf("expected candidate to be added as user, got %+v", addedUser)
	}

	promotedUser := updateAllowedRoleForTest(t, adminClient, server.URL, candidateID, roleAdmin)
	if promotedUser.Role != string(roleAdmin) {
		t.Fatalf("expected candidate to become admin, got %+v", promotedUser)
	}

	assertJSONStatus(t, adminClient, http.MethodDelete, server.URL+"/api/admin/allowed-users/"+strconv.FormatInt(candidateID, 10), nil, http.StatusConflict)

	demotedUser := updateAllowedRoleForTest(t, adminClient, server.URL, candidateID, roleUser)
	if demotedUser.Role != string(roleUser) {
		t.Fatalf("expected candidate to be demoted to user, got %+v", demotedUser)
	}

	deleteAllowedUserForTest(t, adminClient, server.URL, candidateID)
	usersAfterDelete := fetchAdminUsers(t, adminClient, server.URL)
	if findAdminUser(usersAfterDelete, candidateID).IsAllowed {
		t.Fatalf("expected candidate to be removed from allowlist")
	}

	likes := fetchAdminLikes(t, adminClient, server.URL)
	postLikes := findAdminPostLikes(likes, post.ID)
	if postLikes.PostID == 0 {
		t.Fatalf("expected admin likes view to include post %d", post.ID)
	}
	if postLikes.LikeCount != 1 {
		t.Fatalf("expected 1 like in admin likes view, got %+v", postLikes)
	}
	if len(postLikes.LikedUsers) != 1 || postLikes.LikedUsers[0].Username != "liker" {
		t.Fatalf("expected liker to appear in admin likes view, got %+v", postLikes)
	}
}

func TestAdminCannotDemoteLastAdmin(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	adminClient := newSessionClient(t)
	registerUser(t, adminClient, server.URL, "solo", "secret12")
	allowUserByUsernameAs(t, "solo", roleAdmin)
	soloID := userIDByUsername(t, "solo")

	var response apiResponse
	status := requestJSONStatus(
		t,
		adminClient,
		http.MethodPatch,
		server.URL+"/api/admin/allowed-users/"+strconv.FormatInt(soloID, 10)+"/role",
		map[string]string{"role": string(roleUser)},
		&response,
	)
	if status != http.StatusConflict {
		t.Fatalf("expected last-admin demotion to return 409, got %d with %+v", status, response)
	}
	if !strings.Contains(response.Message, "последнего администратора") {
		t.Fatalf("expected last-admin guard message, got %q", response.Message)
	}
}

func TestAdminRouteServesPage(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := newSessionClient(t)
	body, status := getBinary(t, client, server.URL+"/admin")
	if status != http.StatusOK {
		t.Fatalf("expected /admin to return 200, got %d", status)
	}
	if !bytes.Contains(body, []byte("Админка Кузовка")) {
		t.Fatalf("expected /admin page to contain admin title")
	}
}

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	rootDir := t.TempDir()
	testDB := openTestDB(t, filepath.Join(rootDir, "kusovok-test.db"))

	previousDB := db
	previousCookiePath := cookiePath
	previousSecureCookies := secureCookies
	previousImageDirPath := imageDirPath

	db = testDB
	cookiePath = "/"
	secureCookies = false
	imageDirPath = filepath.Join(rootDir, "img")

	t.Cleanup(func() {
		db = previousDB
		cookiePath = previousCookiePath
		secureCookies = previousSecureCookies
		imageDirPath = previousImageDirPath
		_ = testDB.Close()
	})

	if err := initDB(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := ensureImageDir(); err != nil {
		t.Fatalf("ensure image dir: %v", err)
	}

	return httptest.NewServer(newServerMux())
}

func openTestDB(t *testing.T, dbFile string) *sql.DB {
	t.Helper()

	testDB, err := openDB(dbFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return testDB
}

func newSessionClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}

	return &http.Client{Jar: jar}
}

func registerUser(t *testing.T, client *http.Client, baseURL, username, password string) {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodPost, baseURL+"/api/register", map[string]string{
		"username": username,
		"password": password,
	}, &response)
	if !response.Success {
		t.Fatalf("register %s failed: %s", username, response.Message)
	}
}

func createPostForTest(t *testing.T, client *http.Client, baseURL, content string) Post {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodPost, baseURL+"/api/posts", map[string]string{
		"content": content,
	}, &response)
	if !response.Success {
		t.Fatalf("create post failed: %s", response.Message)
	}

	var post Post
	if err := json.Unmarshal(response.Data, &post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	return post
}

func createMultipartPostForTest(t *testing.T, client *http.Client, baseURL, content, fileName string, data []byte) Post {
	t.Helper()

	response := requestMultipart(t, client, baseURL+"/api/posts", content, fileName, data)
	if !response.Success {
		t.Fatalf("create multipart post failed: %s", response.Message)
	}

	var post Post
	if err := json.Unmarshal(response.Data, &post); err != nil {
		t.Fatalf("decode multipart post: %v", err)
	}
	return post
}

func createMultipartPostExpectFailure(t *testing.T, client *http.Client, baseURL, content, fileName string, data []byte) apiResponse {
	t.Helper()

	response := requestMultipart(t, client, baseURL+"/api/posts", content, fileName, data)
	if response.Success {
		t.Fatalf("expected multipart post to fail")
	}
	return response
}

func likePostForTest(t *testing.T, client *http.Client, baseURL string, postID int64) likeResponse {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodPost, baseURL+"/api/like", map[string]int64{
		"post_id": postID,
	}, &response)
	if !response.Success {
		t.Fatalf("like post failed: %s", response.Message)
	}

	var result likeResponse
	if err := json.Unmarshal(response.Data, &result); err != nil {
		t.Fatalf("decode like response: %v", err)
	}
	return result
}

func fetchFeed(t *testing.T, client *http.Client, baseURL string) []Post {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodGet, baseURL+"/api/feed", nil, &response)
	if !response.Success {
		t.Fatalf("fetch feed failed: %s", response.Message)
	}

	var posts []Post
	if err := json.Unmarshal(response.Data, &posts); err != nil {
		t.Fatalf("decode feed: %v", err)
	}
	return posts
}

func requestJSON(t *testing.T, client *http.Client, method, url string, payload interface{}, target interface{}) {
	t.Helper()

	_ = requestJSONStatus(t, client, method, url, payload, target)
}

func requestJSONStatus(t *testing.T, client *http.Client, method, url string, payload interface{}, target interface{}) int {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode
}

func requestMultipart(t *testing.T, client *http.Client, url, content, fileName string, data []byte) apiResponse {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("content", content); err != nil {
		t.Fatalf("write content field: %v", err)
	}
	if data != nil {
		fileWriter, err := writer.CreateFormFile("image", fileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fileWriter.Write(data); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("new multipart request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do multipart request: %v", err)
	}
	defer resp.Body.Close()

	var response apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode multipart response: %v", err)
	}
	return response
}

func getBinary(t *testing.T, client *http.Client, url string) ([]byte, int) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new binary request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do binary request: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read binary response: %v", err)
	}
	return data, resp.StatusCode
}

func fetchMe(t *testing.T, client *http.Client, baseURL string) MeResponse {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodGet, baseURL+"/api/me", nil, &response)
	if !response.Success {
		t.Fatalf("fetch me failed: %s", response.Message)
	}

	var me MeResponse
	if err := json.Unmarshal(response.Data, &me); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	return me
}

func allowUserByUsername(t *testing.T, username string) {
	t.Helper()

	allowUserByUsernameAs(t, username, roleUser)
}

func allowUserByUsernameAs(t *testing.T, username string, role AllowedRole) {
	t.Helper()

	userID := userIDByUsername(t, username)

	if _, err := db.Exec("INSERT INTO allowed_users (user_id, role) VALUES (?, ?)", userID, string(role)); err != nil {
		t.Fatalf("allow user %s: %v", username, err)
	}
}

func userIDByUsername(t *testing.T, username string) int64 {
	t.Helper()

	var userID int64
	if err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID); err != nil {
		t.Fatalf("find user %s: %v", username, err)
	}
	return userID
}

func assertForbiddenJSON(t *testing.T, client *http.Client, method, url string, payload interface{}) {
	t.Helper()

	var response apiResponse
	status := requestJSONStatus(t, client, method, url, payload, &response)
	if status != http.StatusForbidden {
		t.Fatalf("expected forbidden status for %s %s, got %d with response %+v", method, url, status, response)
	}
	if response.Success {
		t.Fatalf("expected forbidden response to fail, got %+v", response)
	}
	if response.Message != accessDeniedMessage {
		t.Fatalf("expected forbidden message %q, got %q", accessDeniedMessage, response.Message)
	}
}

func assertJSONStatus(t *testing.T, client *http.Client, method, url string, payload interface{}, expected int) apiResponse {
	t.Helper()

	var response apiResponse
	status := requestJSONStatus(t, client, method, url, payload, &response)
	if status != expected {
		t.Fatalf("expected status %d for %s %s, got %d with response %+v", expected, method, url, status, response)
	}
	return response
}

func fetchAdminUsers(t *testing.T, client *http.Client, baseURL string) []AdminUserSummary {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodGet, baseURL+"/api/admin/users", nil, &response)
	if !response.Success {
		t.Fatalf("fetch admin users failed: %s", response.Message)
	}

	var users []AdminUserSummary
	if err := json.Unmarshal(response.Data, &users); err != nil {
		t.Fatalf("decode admin users: %v", err)
	}
	return users
}

func fetchAdminLikes(t *testing.T, client *http.Client, baseURL string) []AdminPostLikes {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodGet, baseURL+"/api/admin/likes", nil, &response)
	if !response.Success {
		t.Fatalf("fetch admin likes failed: %s", response.Message)
	}

	var likes []AdminPostLikes
	if err := json.Unmarshal(response.Data, &likes); err != nil {
		t.Fatalf("decode admin likes: %v", err)
	}
	return likes
}

func addAllowedUserForTest(t *testing.T, client *http.Client, baseURL string, userID int64) AdminUserSummary {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodPost, baseURL+"/api/admin/allowed-users", map[string]int64{
		"user_id": userID,
	}, &response)
	if !response.Success {
		t.Fatalf("add allowed user failed: %s", response.Message)
	}

	var user AdminUserSummary
	if err := json.Unmarshal(response.Data, &user); err != nil {
		t.Fatalf("decode allowed user response: %v", err)
	}
	return user
}

func updateAllowedRoleForTest(t *testing.T, client *http.Client, baseURL string, userID int64, role AllowedRole) AdminUserSummary {
	t.Helper()

	var response apiResponse
	requestJSON(
		t,
		client,
		http.MethodPatch,
		baseURL+"/api/admin/allowed-users/"+strconv.FormatInt(userID, 10)+"/role",
		map[string]string{"role": string(role)},
		&response,
	)
	if !response.Success {
		t.Fatalf("update allowed role failed: %s", response.Message)
	}

	var user AdminUserSummary
	if err := json.Unmarshal(response.Data, &user); err != nil {
		t.Fatalf("decode updated role response: %v", err)
	}
	return user
}

func deleteAllowedUserForTest(t *testing.T, client *http.Client, baseURL string, userID int64) {
	t.Helper()

	var response apiResponse
	requestJSON(t, client, http.MethodDelete, baseURL+"/api/admin/allowed-users/"+strconv.FormatInt(userID, 10), nil, &response)
	if !response.Success {
		t.Fatalf("delete allowed user failed: %s", response.Message)
	}
}

func findAdminUser(users []AdminUserSummary, userID int64) AdminUserSummary {
	for _, user := range users {
		if user.ID == userID {
			return user
		}
	}
	return AdminUserSummary{}
}

func findAdminPostLikes(posts []AdminPostLikes, postID int64) AdminPostLikes {
	for _, post := range posts {
		if post.PostID == postID {
			return post
		}
	}
	return AdminPostLikes{}
}

func tableColumns(t *testing.T, database *sql.DB, table string) map[string]bool {
	t.Helper()

	rows, err := database.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("table info query: %v", err)
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
			t.Fatalf("scan table info: %v", err)
		}
		columns[name] = true
	}
	return columns
}

func hasIndex(t *testing.T, database *sql.DB, table, index string) bool {
	t.Helper()

	rows, err := database.Query("PRAGMA index_list(" + table + ")")
	if err != nil {
		t.Fatalf("index list query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index list: %v", err)
		}
		if name == index {
			return true
		}
	}
	return false
}

func findPostByID(posts []Post, postID int64) Post {
	for _, post := range posts {
		if post.ID == postID {
			return post
		}
	}
	return Post{}
}

func assertImageDirEmpty(t *testing.T) {
	t.Helper()

	entries, err := os.ReadDir(imageDirPath)
	if err != nil {
		t.Fatalf("read image dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected image dir to be empty, got %d file(s)", len(entries))
	}
}

func pngFixture() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB1, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}
