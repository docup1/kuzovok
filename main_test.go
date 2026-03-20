package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
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

func TestRegisterCreatePostAndToggleLike(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	alice := newSessionClient(t)
	registerUser(t, alice, server.URL, "alice", "secret12")
	post := createPostForTest(t, alice, server.URL, "Первый локальный пост")

	bob := newSessionClient(t)
	registerUser(t, bob, server.URL, "bob", "secret12")

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

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	previousCookiePath := cookiePath
	previousSecureCookies := secureCookies
	cookiePath = "/"
	secureCookies = false
	t.Cleanup(func() {
		cookiePath = previousCookiePath
		secureCookies = previousSecureCookies
	})

	testDB, err := openDB(filepath.Join(t.TempDir(), "kusovok-test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db = testDB
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := initDB(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	return httptest.NewServer(newServerMux())
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
}
