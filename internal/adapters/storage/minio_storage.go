package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"adapter/internal/ports"
)

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

// MinIOStorage implements ports.ObjectStorage using MinIO.
type MinIOStorage struct {
	client *minio.Client
	cfg    MinIOConfig
}

func NewMinIOStorage(cfg MinIOConfig) (ports.ObjectStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		cfg:    cfg,
	}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)

	_, err := s.client.PutObject(ctx, s.cfg.Bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	// Return the object key (full path) instead of URL
	return objectName, nil
}

func (s *MinIOStorage) GetBucket() string {
	return s.cfg.Bucket
}
