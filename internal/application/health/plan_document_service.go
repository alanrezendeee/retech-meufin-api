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

// planDocAllowedMimes aceita PDFs, imagens de celular e documentos de texto.
var planDocAllowedMimes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/heic":         true,
	"image/heif":         true,
	"image/webp":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var planDocUnsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

const planDocBucket = "health"

// PlanDocumentService orquestra upload/listagem/download de documentos de um
// plano de saúde (contrato/apólice, carteirinha, boletos etc.) sobre o storage.
type PlanDocumentService struct {
	repo           dom.PlanDocumentRepository
	planRepo       dom.PlanRepository
	storage        storage.ObjectStorage
	maxUploadBytes int64
}

// NewPlanDocumentService cria o serviço. maxUploadMB <= 0 usa o default (20MB).
func NewPlanDocumentService(repo dom.PlanDocumentRepository, planRepo dom.PlanRepository, st storage.ObjectStorage, maxUploadMB int) *PlanDocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &PlanDocumentService{
		repo:           repo,
		planRepo:       planRepo,
		storage:        st,
		maxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
}

type UploadPlanDocInput struct {
	WorkspaceID      uuid.UUID
	PlanID           uuid.UUID
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

func (s *PlanDocumentService) Upload(ctx context.Context, in UploadPlanDocInput) (*dom.PlanDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	// Garante que o plano existe neste workspace.
	if _, err := s.planRepo.GetByID(ctx, in.WorkspaceID, in.PlanID); err != nil {
		return nil, err
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !planDocAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (PDF, imagem ou DOC)"}
	}

	safeName := sanitizePlanDocFileName(in.OriginalFileName)
	objectKey := buildPlanDocObjectKey(in.WorkspaceID, in.PlanID, safeName)

	now := time.Now().UTC()
	doc := &dom.PlanDocument{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		PlanID:           in.PlanID,
		DocType:          dom.PlanDocType(strings.TrimSpace(strings.ToLower(in.DocType))),
		Label:            in.Label,
		DocNumber:        in.DocNumber,
		ValidUntil:       in.ValidUntil,
		Notes:            in.Notes,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		MimeType:         mime,
		SizeBytes:        in.Size,
		StorageProvider:  "minio",
		Bucket:           planDocBucket,
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

type ListPlanDocumentsResult struct {
	Items []dom.PlanDocument
	Total int64
}

func (s *PlanDocumentService) ListByPlan(ctx context.Context, workspaceID, planID uuid.UUID, limit, offset int) (*ListPlanDocumentsResult, error) {
	items, total, err := s.repo.ListByPlan(ctx, workspaceID, planID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListPlanDocumentsResult{Items: items, Total: total}, nil
}

func (s *PlanDocumentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// DownloadURL gera uma URL presignada de download (validade 5 minutos).
func (s *PlanDocumentService) DownloadURL(ctx context.Context, workspaceID, id uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return "", err
	}
	if !s.storage.Enabled() {
		return "", &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	return s.storage.PresignedGetURL(ctx, doc.ObjectKey, 5*time.Minute)
}

func sanitizePlanDocFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = planDocUnsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildPlanDocObjectKey(workspaceID, planID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/health/plans/%s/docs/%04d/%02d/%s-%s",
		workspaceID.String(), planID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
