package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStorage implementa ObjectStorage sobre um servidor MinIO/S3 compatível.
type MinioStorage struct {
	client *minio.Client
	bucket string
}

// NewMinioStorage cria o client MinIO e garante que o bucket exista.
func NewMinioStorage(cfg Config) (*MinioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: falha ao criar client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: falha ao verificar bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio: falha ao criar bucket %q: %w", cfg.Bucket, err)
		}
	}

	return &MinioStorage{client: client, bucket: cfg.Bucket}, nil
}

// Enabled sempre retorna true para o storage MinIO ativo.
func (s *MinioStorage) Enabled() bool { return true }

// Put envia o objeto para o bucket configurado.
func (s *MinioStorage) Put(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio: falha no upload de %q: %w", objectKey, err)
	}
	return nil
}

// Get retorna o conteúdo do objeto.
func (s *MinioStorage) Get(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: falha ao ler %q: %w", objectKey, err)
	}
	return obj, nil
}

// PresignedGetURL gera uma URL temporária de download para o objeto.
func (s *MinioStorage) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("minio: falha ao gerar presigned URL de %q: %w", objectKey, err)
	}
	return u.String(), nil
}
