package storage

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type localFileStore struct {
	uploadDir string
	baseURL   string
}

func NewLocalFileStore(uploadDir, baseURL string) (FileStore, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return &localFileStore{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}, nil
}

func (s *localFileStore) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	filePath := filepath.Join(s.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.baseURL, filename), nil
}

func (s *localFileStore) DeleteFile(ctx context.Context, fileURL string) error {
	filename := filepath.Base(fileURL)
	filePath := filepath.Join(s.uploadDir, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
