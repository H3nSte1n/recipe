package storage

import (
	"context"
	"mime/multipart"
)

type FileStore interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}
