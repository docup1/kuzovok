package storage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidImageURL = errors.New("invalid image url")
	ErrInvalidFilename = errors.New("invalid image file name")
	ErrPathTraversal   = errors.New("invalid image path")
)

type ImageStorage struct {
	dir          string
	publicPrefix string
	allowedTypes map[string]string
	maxSize      int64
}

func NewImageStorage(dir, publicPrefix string, allowedTypes []string, maxSizeMB int) *ImageStorage {
	allowed := make(map[string]string)
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	for _, t := range allowedTypes {
		if ext, ok := extensions[t]; ok {
			allowed[t] = ext
		}
	}
	return &ImageStorage{
		dir:          dir,
		publicPrefix: publicPrefix,
		allowedTypes: allowed,
		maxSize:      int64(maxSizeMB) << 20,
	}
}

func (s *ImageStorage) Save(data []byte, contentType string) (url string, expiresAt string, err error) {
	ext, ok := s.allowedTypes[contentType]
	if !ok {
		return "", "", errors.New("invalid content type")
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", "", err
	}

	filename := uuid.NewString() + ext
	filePath := filepath.Join(s.dir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", "", err
	}

	return path.Join(s.publicPrefix, filename), path.Join(s.publicPrefix, filename), nil
}

func (s *ImageStorage) Delete(imageURL string) error {
	filePath, err := s.ResolvePath(imageURL)
	if err != nil {
		return err
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *ImageStorage) ResolvePath(imageURL string) (string, error) {
	filename := strings.TrimPrefix(imageURL, s.publicPrefix)
	if filename == imageURL || filename == "" {
		return "", ErrInvalidImageURL
	}
	if filename != path.Base(filename) {
		return "", ErrInvalidFilename
	}

	cleanDir := filepath.Clean(s.dir)
	filePath := filepath.Clean(filepath.Join(cleanDir, filename))
	prefix := cleanDir + string(os.PathSeparator)
	if filePath != cleanDir && !strings.HasPrefix(filePath, prefix) {
		return "", ErrPathTraversal
	}
	return filePath, nil
}

func (s *ImageStorage) Exists(imageURL string) bool {
	filePath, err := s.ResolvePath(imageURL)
	if err != nil {
		return false
	}
	_, err = os.Stat(filePath)
	return err == nil
}

func (s *ImageStorage) GetMaxSize() int64 {
	return s.maxSize
}

func (s *ImageStorage) GetAllowedTypes() map[string]string {
	return s.allowedTypes
}

func (s *ImageStorage) ValidateSize(data []byte) error {
	if int64(len(data)) > s.maxSize {
		return fmt.Errorf("image too large, max %d bytes", s.maxSize)
	}
	return nil
}

func (s *ImageStorage) ValidateType(contentType string) bool {
	_, ok := s.allowedTypes[contentType]
	return ok
}
