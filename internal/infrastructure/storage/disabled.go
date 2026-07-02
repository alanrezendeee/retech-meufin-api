package storage

import (
	"context"
	"errors"
	"io"
	"log"
	"time"
)

// ErrStorageDisabled é retornado quando o storage de objetos não está configurado.
var ErrStorageDisabled = errors.New("storage de objetos não configurado")

// DisabledStorage é um fallback que não persiste nada; usado quando MinIO não está configurado.
type DisabledStorage struct{}

// Enabled sempre retorna false.
func (DisabledStorage) Enabled() bool { return false }

// Put retorna ErrStorageDisabled.
func (DisabledStorage) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return ErrStorageDisabled
}

// PresignedGetURL retorna ErrStorageDisabled.
func (DisabledStorage) PresignedGetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", ErrStorageDisabled
}

// Get retorna ErrStorageDisabled.
func (DisabledStorage) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, ErrStorageDisabled
}

// New retorna um MinioStorage quando a configuracao esta completa e valida;
// caso contrário (ou se a construção falhar) retorna DisabledStorage para que a API não quebre.
func New(cfg Config) ObjectStorage {
	// Config incompleta não loga aqui: o main loga o estado de todas as
	// conexões no boot (✅/⚠️) com a lista de envs esperadas.
	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return DisabledStorage{}
	}
	s, err := NewMinioStorage(cfg)
	if err != nil {
		log.Printf("storage: falha ao inicializar MinIO (%v); usando storage desabilitado", err)
		return DisabledStorage{}
	}
	return s
}
