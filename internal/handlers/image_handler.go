package handlers

import (
	"net/http"

	"kusovok/internal/infrastructure/storage"
)

type ImageHandler struct {
	storage *storage.ImageStorage
}

func NewImageHandler(storage *storage.ImageStorage) *ImageHandler {
	return &ImageHandler{storage: storage}
}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filePath, err := h.storage.ResolvePath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !h.storage.Exists(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	http.ServeFile(w, r, filePath)
}
