package storage

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ObjectStorage abstrai o backend de objetos (MinIO/S3) usado por documentos de saúde.
type ObjectStorage interface {
	// Enabled indica se o storage está configurado e operacional.
	Enabled() bool
	// Put envia um objeto para o storage.
	Put(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) error
	// PresignedGetURL gera uma URL temporária de download.
	PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	// Get retorna o conteúdo do objeto (para extração). O chamador deve fechar.
	Get(ctx context.Context, objectKey string) (io.ReadCloser, error)
}

// Config carrega a configuração do storage de objetos.
type Config struct {
	Endpoint    string
	AccessKey   string
	SecretKey   string
	Bucket      string
	UseSSL      bool
	MaxUploadMB int
}

const defaultMaxUploadMB = 20

// ConfigFromEnv lê a configuração do storage a partir das variáveis de ambiente.
//
//	MINIO_ENDPOINT      host:port do MinIO/S3
//	MINIO_ACCESS_KEY    access key
//	MINIO_SECRET_KEY    secret key
//	MINIO_BUCKET_HEALTH bucket de documentos de saúde
//	MINIO_USE_SSL       true/false (default false)
//	HEALTH_MAX_UPLOAD_MB tamanho máximo de upload em MB (default 20)
func ConfigFromEnv() Config {
	maxMB := defaultMaxUploadMB
	if v := strings.TrimSpace(os.Getenv("HEALTH_MAX_UPLOAD_MB")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxMB = n
		}
	}
	return Config{
		Endpoint:    strings.TrimSpace(os.Getenv("MINIO_ENDPOINT")),
		AccessKey:   strings.TrimSpace(os.Getenv("MINIO_ACCESS_KEY")),
		SecretKey:   strings.TrimSpace(os.Getenv("MINIO_SECRET_KEY")),
		Bucket:      strings.TrimSpace(os.Getenv("MINIO_BUCKET_HEALTH")),
		UseSSL:      strings.EqualFold(strings.TrimSpace(os.Getenv("MINIO_USE_SSL")), "true"),
		MaxUploadMB: maxMB,
	}
}
