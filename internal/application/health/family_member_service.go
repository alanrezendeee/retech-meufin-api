package health

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// memberAvatarAllowedMimes restringe o avatar a formatos de imagem web comuns.
var memberAvatarAllowedMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

const (
	memberAvatarMaxBytes = 5 * 1024 * 1024
	memberAvatarTTL      = 15 * time.Minute
)

type FamilyMemberService struct {
	repo    dom.FamilyMemberRepository
	storage storage.ObjectStorage
}

// NewFamilyMemberService cria o serviço. O object storage é opcional (variádico)
// para não quebrar chamadas existentes: passe-o para habilitar avatares de membro.
func NewFamilyMemberService(repo dom.FamilyMemberRepository, st ...storage.ObjectStorage) *FamilyMemberService {
	var s storage.ObjectStorage
	if len(st) > 0 {
		s = st[0]
	}
	return &FamilyMemberService{repo: repo, storage: s}
}

type CreateFamilyMemberInput struct {
	WorkspaceID  uuid.UUID
	FullName     string
	Relationship string
	BirthDate    *time.Time
	Gender       *string
	Document     *string
	Notes        *string
	HeightCm     *float64
	WeightKg     *float64
	Active       *bool
}

type UpdateFamilyMemberInput struct {
	WorkspaceID  uuid.UUID
	ID           uuid.UUID
	FullName     string
	Relationship string
	BirthDate    *time.Time
	Gender       *string
	Document     *string
	Notes        *string
	HeightCm     *float64
	WeightKg     *float64
	Active       *bool
}

func (s *FamilyMemberService) Create(ctx context.Context, in CreateFamilyMemberInput) (*dom.FamilyMember, error) {
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	f := &dom.FamilyMember{
		ID:           uuid.New(),
		WorkspaceID:  in.WorkspaceID,
		FullName:     in.FullName,
		Relationship: in.Relationship,
		BirthDate:    in.BirthDate,
		Gender:       in.Gender,
		Document:     in.Document,
		Notes:        in.Notes,
		HeightCm:     in.HeightCm,
		WeightKg:     in.WeightKg,
		Active:       active,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FamilyMemberService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FamilyMember, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *FamilyMemberService) Update(ctx context.Context, in UpdateFamilyMemberInput) (*dom.FamilyMember, error) {
	f, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	f.FullName = in.FullName
	f.Relationship = in.Relationship
	f.BirthDate = in.BirthDate
	f.Gender = in.Gender
	f.Document = in.Document
	f.Notes = in.Notes
	f.HeightCm = in.HeightCm
	f.WeightKg = in.WeightKg
	if in.Active != nil {
		f.Active = *in.Active
	}
	f.UpdatedAt = time.Now().UTC()
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FamilyMemberService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

type ListFamilyMembersResult struct {
	Items []dom.FamilyMember
	Total int64
}

func (s *FamilyMemberService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FamilyMemberFilter, limit, offset int) (*ListFamilyMembersResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListFamilyMembersResult{Items: items, Total: total}, nil
}

// Birthdays retorna os membros ativos com data de nascimento, ordenados pelo
// próximo aniversário (menos dias restantes primeiro).
func (s *FamilyMemberService) Birthdays(ctx context.Context, workspaceID uuid.UUID) ([]dom.Birthday, error) {
	members, err := s.repo.ListWithBirthDate(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]dom.Birthday, 0, len(members))
	for i := range members {
		b, ok := dom.NextBirthdayOf(members[i], now)
		if !ok {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DaysUntil != out[j].DaysUntil {
			return out[i].DaysUntil < out[j].DaysUntil
		}
		return out[i].Member.FullName < out[j].Member.FullName
	})
	return out, nil
}

// SetAvatarInput carrega a foto enviada para um membro da família.
type SetAvatarInput struct {
	WorkspaceID uuid.UUID
	MemberID    uuid.UUID
	MimeType    string
	Size        int64
	Content     io.Reader
}

// SetAvatar valida e envia a foto do membro ao storage, gravando a object key.
// A key é fixa por membro (avatar.jpg), então novos uploads sobrescrevem o anterior.
func (s *FamilyMemberService) SetAvatar(ctx context.Context, in SetAvatarInput) (*dom.FamilyMember, error) {
	if s.storage == nil || !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de fotos indisponível (storage não configurado)"}
	}
	member, err := s.repo.GetByID(ctx, in.WorkspaceID, in.MemberID)
	if err != nil {
		return nil, err
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > memberAvatarMaxBytes {
		return nil, &dom.ValidationError{Msg: "imagem excede o limite de 5 MB"}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !memberAvatarAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "formato inválido (use JPEG, PNG ou WEBP)"}
	}

	key := buildMemberAvatarObjectKey(in.WorkspaceID, in.MemberID)
	if err := s.storage.Put(ctx, key, in.Content, in.Size, mime); err != nil {
		return nil, fmt.Errorf("falha ao enviar foto: %w", err)
	}
	if err := s.repo.UpdateAvatar(ctx, in.WorkspaceID, in.MemberID, &key); err != nil {
		return nil, err
	}
	member.AvatarObjectKey = &key
	return member, nil
}

// RemoveAvatar limpa a foto do membro (não apaga o objeto do storage).
func (s *FamilyMemberService) RemoveAvatar(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.UpdateAvatar(ctx, workspaceID, id, nil)
}

// AvatarURL gera uma URL presignada (15min) da foto, ou nil se não houver foto
// ou o storage estiver indisponível. Best-effort: nunca propaga erro.
func (s *FamilyMemberService) AvatarURL(ctx context.Context, key *string) *string {
	if key == nil || strings.TrimSpace(*key) == "" {
		return nil
	}
	if s.storage == nil || !s.storage.Enabled() {
		return nil
	}
	url, err := s.storage.PresignedGetURL(ctx, *key, memberAvatarTTL)
	if err != nil || url == "" {
		return nil
	}
	return &url
}

func buildMemberAvatarObjectKey(workspaceID, memberID uuid.UUID) string {
	return fmt.Sprintf("tenants/%s/members/%s/avatar.jpg", workspaceID.String(), memberID.String())
}

func IsNotFound(err error) bool {
	return errors.Is(err, dom.ErrNotFound)
}
