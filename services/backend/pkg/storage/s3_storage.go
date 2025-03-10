package storage

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"mime/multipart"
)

type s3FileStore struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3FileStore(client *s3.Client, bucket, region string) FileStore {
	return &s3FileStore{
		client: client,
		bucket: bucket,
		region: region,
	}
}

func (s *s3FileStore) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// Implementation for S3
	return "", nil
}

func (s *s3FileStore) DeleteFile(ctx context.Context, fileURL string) error {
	// Implementation for S3
	return nil
}
