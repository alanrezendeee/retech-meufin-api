package warranty

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/warranty"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// warrantyDocAllowedMimes aceita PDFs, imagens (foto da nota) e documentos.
var warrantyDocAllowedMimes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/heic":         true,
	"image/heif":         true,
	"image/webp":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var warrantyDocUnsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// DocumentService orquestra upload/listagem/download de documentos de garantias
// sobre o object storage.
type DocumentService struct {
	repo           dom.Repository
	storage        storage.ObjectStorage
	maxUploadBytes int64
}

// NewDocumentService cria o serviço. maxUploadMB <= 0 usa o default (20MB).
func NewDocumentService(repo dom.Repository, st storage.ObjectStorage, maxUploadMB int) *DocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &DocumentService{
		repo:           repo,
		storage:        st,
		maxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
}

type UploadDocumentInput struct {
	WorkspaceID      uuid.UUID
	WarrantyID       uuid.UUID
	DocType          string
	Notes            *string
	OriginalFileName string
	ContentType      string
	Size             int64
	Content          io.Reader
}

func (s *DocumentService) Upload(ctx context.Context, in UploadDocumentInput) (*dom.Document, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	// Garante que a garantia existe neste workspace.
	if _, err := s.repo.GetByID(ctx, in.WorkspaceID, in.WarrantyID); err != nil {
		return nil, err
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.ContentType))
	if !warrantyDocAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (PDF, imagem ou DOC)"}
	}
	docType := dom.DocType(strings.TrimSpace(strings.ToLower(in.DocType)))
	if docType == "" {
		docType = dom.DocOutros
	}

	safeName := sanitizeWarrantyDocFileName(in.OriginalFileName)
	objectKey := buildWarrantyDocObjectKey(in.WorkspaceID, in.WarrantyID, safeName)

	doc := &dom.Document{
		ID:               uuid.New(),
		WarrantyID:       in.WarrantyID,
		WorkspaceID:      in.WorkspaceID,
		DocType:          docType,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		ObjectKey:        objectKey,
		ContentType:      mime,
		SizeBytes:        in.Size,
		Notes:            in.Notes,
		CreatedAt:        time.Now().UTC(),
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	if err := s.storage.Put(ctx, objectKey, in.Content, in.Size, doc.ContentType); err != nil {
		return nil, fmt.Errorf("falha ao enviar arquivo: %w", err)
	}
	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) List(ctx context.Context, workspaceID, warrantyID uuid.UUID) ([]dom.Document, error) {
	// Garante que a garantia existe neste workspace.
	if _, err := s.repo.GetByID(ctx, workspaceID, warrantyID); err != nil {
		return nil, err
	}
	return s.repo.ListDocuments(ctx, workspaceID, warrantyID)
}

// DownloadURL gera uma URL presignada de download (validade 5 minutos).
func (s *DocumentService) DownloadURL(ctx context.Context, workspaceID, docID uuid.UUID) (string, error) {
	doc, err := s.repo.GetDocumentByID(ctx, workspaceID, docID)
	if err != nil {
		return "", err
	}
	if !s.storage.Enabled() {
		return "", &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	return s.storage.PresignedGetURL(ctx, doc.ObjectKey, 5*time.Minute)
}

func (s *DocumentService) Delete(ctx context.Context, workspaceID, docID uuid.UUID) error {
	return s.repo.DeleteDocument(ctx, workspaceID, docID)
}

func sanitizeWarrantyDocFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = warrantyDocUnsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildWarrantyDocObjectKey(workspaceID, warrantyID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/warranties/%s/docs/%04d/%02d/%s-%s",
		workspaceID.String(), warrantyID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
