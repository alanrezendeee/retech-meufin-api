package health

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// memberDocAllowedMimes aceita fotos de celular, PDFs e documentos de texto.
var memberDocAllowedMimes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/heic":         true,
	"image/heif":         true,
	"image/webp":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var memberDocUnsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

const memberDocBucket = "health"

// MemberDocumentService orquestra upload/listagem/download de documentos
// pessoais dos membros da família sobre o object storage.
type MemberDocumentService struct {
	repo           dom.MemberDocumentRepository
	memberRepo     dom.FamilyMemberRepository
	storage        storage.ObjectStorage
	maxUploadBytes int64
}

// NewMemberDocumentService cria o serviço. maxUploadMB <= 0 usa o default (20MB).
func NewMemberDocumentService(repo dom.MemberDocumentRepository, memberRepo dom.FamilyMemberRepository, st storage.ObjectStorage, maxUploadMB int) *MemberDocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &MemberDocumentService{
		repo:           repo,
		memberRepo:     memberRepo,
		storage:        st,
		maxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
}

type UploadMemberDocInput struct {
	WorkspaceID      uuid.UUID
	FamilyMemberID   uuid.UUID
	UploadedByUserID uuid.UUID
	DocType          string
	Label            *string
	DocNumber        *string
	ValidUntil       *time.Time
	Notes            *string
	OriginalFileName string
	MimeType         string
	Size             int64
	Content          io.Reader
}

func (s *MemberDocumentService) Upload(ctx context.Context, in UploadMemberDocInput) (*dom.MemberDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	// Garante que o membro existe neste workspace.
	if _, err := s.memberRepo.GetByID(ctx, in.WorkspaceID, in.FamilyMemberID); err != nil {
		return nil, err
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !memberDocAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (PDF, imagem ou DOC)"}
	}

	safeName := sanitizeMemberDocFileName(in.OriginalFileName)
	objectKey := buildMemberDocObjectKey(in.WorkspaceID, in.FamilyMemberID, safeName)

	now := time.Now().UTC()
	doc := &dom.MemberDocument{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		FamilyMemberID:   in.FamilyMemberID,
		DocType:          dom.MemberDocType(strings.TrimSpace(strings.ToLower(in.DocType))),
		Label:            in.Label,
		DocNumber:        in.DocNumber,
		ValidUntil:       in.ValidUntil,
		Notes:            in.Notes,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		MimeType:         mime,
		SizeBytes:        in.Size,
		StorageProvider:  "minio",
		Bucket:           memberDocBucket,
		ObjectKey:        objectKey,
		UploadedByUserID: in.UploadedByUserID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	if err := s.storage.Put(ctx, objectKey, in.Content, in.Size, doc.MimeType); err != nil {
		return nil, fmt.Errorf("falha ao enviar arquivo: %w", err)
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

type ListMemberDocumentsResult struct {
	Items []dom.MemberDocument
	Total int64
}

func (s *MemberDocumentService) ListByMember(ctx context.Context, workspaceID, familyMemberID uuid.UUID, limit, offset int) (*ListMemberDocumentsResult, error) {
	items, total, err := s.repo.ListByMember(ctx, workspaceID, familyMemberID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListMemberDocumentsResult{Items: items, Total: total}, nil
}

func (s *MemberDocumentService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.MemberDocument, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *MemberDocumentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// DownloadURL gera uma URL presignada de download (validade 5 minutos).
func (s *MemberDocumentService) DownloadURL(ctx context.Context, workspaceID, id uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return "", err
	}
	if !s.storage.Enabled() {
		return "", &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	return s.storage.PresignedGetURL(ctx, doc.ObjectKey, 5*time.Minute)
}

func sanitizeMemberDocFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = memberDocUnsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildMemberDocObjectKey(workspaceID, memberID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/health/members/%s/docs/%04d/%02d/%s-%s",
		workspaceID.String(), memberID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
