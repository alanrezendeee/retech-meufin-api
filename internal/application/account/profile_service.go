package account

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// ErrProfileValidation sinaliza entrada inválida no upload de avatar.
type ErrProfileValidation struct{ Msg string }

func (e *ErrProfileValidation) Error() string { return e.Msg }

// avatarAllowedMimes restringe o avatar do usuário a formatos de imagem web comuns.
var avatarAllowedMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

const (
	avatarMaxBytes = 5 * 1024 * 1024
	avatarTTL      = 15 * time.Minute
)

// UserProfile guarda o perfil (1:1) do usuário logado.
type UserProfile struct {
	UserID          uuid.UUID
	WorkspaceID     uuid.UUID
	AvatarObjectKey *string
}

// UserProfileRepository abstrai a persistência do perfil do usuário.
type UserProfileRepository interface {
	// Get retorna o perfil do usuário, ou (nil, nil) se ainda não existir.
	Get(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
	// Upsert cria/atualiza o perfil, gravando a object key do avatar.
	Upsert(ctx context.Context, p *UserProfile) error
	// ClearAvatar limpa a object key do avatar (no-op se não houver perfil).
	ClearAvatar(ctx context.Context, userID uuid.UUID) error
}

// ProfileService orquestra o avatar do usuário logado sobre o object storage.
type ProfileService struct {
	repo    UserProfileRepository
	storage storage.ObjectStorage
}

func NewProfileService(repo UserProfileRepository, st storage.ObjectStorage) *ProfileService {
	return &ProfileService{repo: repo, storage: st}
}

// UploadAvatarInput carrega a foto de perfil enviada pelo usuário logado.
type UploadAvatarInput struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	MimeType    string
	Size        int64
	Content     io.Reader
}

// UploadAvatar valida e envia a foto ao storage e faz upsert no perfil.
// Retorna a URL presignada (15min) da foto recém-enviada.
func (s *ProfileService) UploadAvatar(ctx context.Context, in UploadAvatarInput) (string, error) {
	if s.storage == nil || !s.storage.Enabled() {
		return "", &ErrProfileValidation{Msg: "armazenamento de fotos indisponível (storage não configurado)"}
	}
	if in.UserID == uuid.Nil {
		return "", &ErrProfileValidation{Msg: "usuário inválido"}
	}
	if in.Size <= 0 {
		return "", &ErrProfileValidation{Msg: "arquivo vazio"}
	}
	if in.Size > avatarMaxBytes {
		return "", &ErrProfileValidation{Msg: "imagem excede o limite de 5 MB"}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !avatarAllowedMimes[mime] {
		return "", &ErrProfileValidation{Msg: "formato inválido (use JPEG, PNG ou WEBP)"}
	}

	key := buildUserAvatarObjectKey(in.WorkspaceID, in.UserID)
	if err := s.storage.Put(ctx, key, in.Content, in.Size, mime); err != nil {
		return "", fmt.Errorf("falha ao enviar foto: %w", err)
	}
	if err := s.repo.Upsert(ctx, &UserProfile{
		UserID:          in.UserID,
		WorkspaceID:     in.WorkspaceID,
		AvatarObjectKey: &key,
	}); err != nil {
		return "", err
	}
	url, err := s.storage.PresignedGetURL(ctx, key, avatarTTL)
	if err != nil {
		return "", nil
	}
	return url, nil
}

// AvatarURL retorna a URL presignada (15min) do avatar do usuário, ou "" se não
// houver foto ou o storage estiver indisponível.
func (s *ProfileService) AvatarURL(ctx context.Context, userID uuid.UUID) (string, error) {
	p, err := s.repo.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	if p == nil || p.AvatarObjectKey == nil || strings.TrimSpace(*p.AvatarObjectKey) == "" {
		return "", nil
	}
	if s.storage == nil || !s.storage.Enabled() {
		return "", nil
	}
	url, err := s.storage.PresignedGetURL(ctx, *p.AvatarObjectKey, avatarTTL)
	if err != nil {
		return "", nil
	}
	return url, nil
}

// RemoveAvatar limpa a foto do usuário (não apaga o objeto do storage).
func (s *ProfileService) RemoveAvatar(ctx context.Context, userID uuid.UUID) error {
	return s.repo.ClearAvatar(ctx, userID)
}

// IsProfileValidation informa se o erro é de validação de entrada.
func IsProfileValidation(err error) bool {
	var v *ErrProfileValidation
	return errors.As(err, &v)
}

func buildUserAvatarObjectKey(workspaceID, userID uuid.UUID) string {
	return fmt.Sprintf("tenants/%s/users/%s/avatar.jpg", workspaceID.String(), userID.String())
}
