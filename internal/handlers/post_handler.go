package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"kusovok/internal/application/postapp"
	"kusovok/internal/infrastructure/auth"
	"kusovok/pkg/response"
)

type PostHandler struct {
	createUC    *post.CreatePostUseCase
	feedUC      *post.FeedUseCase
	userPostsUC *post.UserPostsUseCase
	messages    PostMessages
	cfg         PostConfig
}

func NewPostHandler(createUC *post.CreatePostUseCase, feedUC *post.FeedUseCase, userPostsUC *post.UserPostsUseCase, messages PostMessages, cfg PostConfig) *PostHandler {
	return &PostHandler{
		createUC:    createUC,
		feedUC:      feedUC,
		userPostsUC: userPostsUC,
		messages:    messages,
		cfg:         cfg,
	}
}

func (h *PostHandler) Posts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getUserPosts(w, r)
	case http.MethodPost:
		h.createPost(w, r)
	default:
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
	}
}

func (h *PostHandler) getUserPosts(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromRequest(r)

	posts, err := h.userPostsUC.Execute(r.Context(), userID, userID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", posts)
}

func (h *PostHandler) createPost(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromRequest(r)
	username := r.Header.Get("X-Username")

	contentType := r.Header.Get("Content-Type")
	var content string
	var imageData []byte
	var imageContentType string

	if strings.HasPrefix(contentType, "multipart/form-data") {
		var err error
		content, imageData, imageContentType, err = h.parseMultipart(r, w)
		if err != nil {
			response.WriteAppError(w, err)
			return
		}
	} else {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteError(w, http.StatusBadRequest, h.messages.ErrorInvalidData)
			return
		}
		content = req.Content
	}

	post, err := h.createUC.Execute(r.Context(), userID, username, content, imageData, imageContentType)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, h.messages.PostCreated, post)
}

func (h *PostHandler) parseMultipart(r *http.Request, w http.ResponseWriter) (content string, imageData []byte, imageContentType string, err error) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MultipartBodySize)
	if err := r.ParseMultipartForm(h.cfg.MaxImageSize); err != nil {
		if isRequestTooLarge(err) {
			return "", nil, "", response.BadRequest("image too large")
		}
		return "", nil, "", response.BadRequest("invalid form data")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	content = r.FormValue("content")
	file, _, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return content, nil, "", nil
		}
		return "", nil, "", response.BadRequest("failed to read image")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, h.cfg.MaxImageSize+1))
	if err != nil {
		return "", nil, "", response.BadRequest("failed to read image")
	}

	if int64(len(data)) > h.cfg.MaxImageSize {
		return "", nil, "", response.BadRequest("image too large")
	}

	ct := http.DetectContentType(data)
	if !h.isAllowedImageType(ct) {
		return "", nil, "", response.BadRequest("allowed only JPG, PNG, WEBP, GIF")
	}

	return content, data, ct, nil
}

func (h *PostHandler) isAllowedImageType(ct string) bool {
	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	return allowed[ct]
}

func (h *PostHandler) Feed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, h.messages.ErrorInvalidData)
		return
	}

	userID := auth.GetUserIDFromRequest(r)

	posts, err := h.feedUC.Execute(r.Context(), userID, 50)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}

	response.WriteSuccess(w, "", posts)
}

func isRequestTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr) || strings.Contains(strings.ToLower(err.Error()), "request body too large")
}
