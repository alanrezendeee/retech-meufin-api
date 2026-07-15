package patrimony

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// propertyDocAllowedMimes aceita PDFs, imagens de celular e documentos de texto.
var propertyDocAllowedMimes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/heic":         true,
	"image/heif":         true,
	"image/webp":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var propertyDocUnsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// DocumentService orquestra upload/listagem/download de documentos de imóveis
// sobre o object storage.
type DocumentService struct {
	repo         dom.DocumentRepository
	propertyRepo dom.Repository
	storage      storage.ObjectStorage
	maxBytes     int64
}

// NewDocumentService cria o serviço. maxUploadMB <= 0 usa o default (20MB).
func NewDocumentService(repo dom.DocumentRepository, propertyRepo dom.Repository, st storage.ObjectStorage, maxUploadMB int) *DocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &DocumentService{
		repo:         repo,
		propertyRepo: propertyRepo,
		storage:      st,
		maxBytes:     int64(maxUploadMB) * 1024 * 1024,
	}
}

type UploadDocInput struct {
	WorkspaceID      uuid.UUID
	PropertyID       uuid.UUID
	DocType          string
	Notes            *string
	OriginalFileName string
	MimeType         string
	Size             int64
	Content          io.Reader
}

func (s *DocumentService) Upload(ctx context.Context, in UploadDocInput) (*dom.PropertyDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	// Garante que o imóvel existe neste workspace.
	if _, err := s.propertyRepo.GetProperty(ctx, in.WorkspaceID, in.PropertyID); err != nil {
		return nil, err
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !propertyDocAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (PDF, imagem ou DOC)"}
	}

	docType := dom.PropertyDocType(strings.TrimSpace(strings.ToLower(in.DocType)))
	if docType == "" {
		docType = dom.DocOutros
	}
	safeName := sanitizePropertyDocFileName(in.OriginalFileName)
	objectKey := buildPropertyDocObjectKey(in.WorkspaceID, in.PropertyID, safeName)

	doc := &dom.PropertyDocument{
		ID:          uuid.New(),
		PropertyID:  in.PropertyID,
		WorkspaceID: in.WorkspaceID,
		DocType:     docType,
		FileName:    safeName,
		ObjectKey:   objectKey,
		ContentType: mime,
		SizeBytes:   in.Size,
		Notes:       in.Notes,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.storage.Put(ctx, objectKey, in.Content, in.Size, mime); err != nil {
		return nil, fmt.Errorf("falha ao enviar arquivo: %w", err)
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentService) ListByProperty(ctx context.Context, workspaceID, propertyID uuid.UUID) ([]dom.PropertyDocument, error) {
	if _, err := s.propertyRepo.GetProperty(ctx, workspaceID, propertyID); err != nil {
		return nil, err
	}
	return s.repo.ListByProperty(ctx, workspaceID, propertyID)
}

func (s *DocumentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.Delete(ctx, workspaceID, id)
}

// DownloadURL gera uma URL presignada de download (validade 5 minutos).
func (s *DocumentService) DownloadURL(ctx context.Context, workspaceID, id uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return "", err
	}
	if !s.storage.Enabled() {
		return "", &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	return s.storage.PresignedGetURL(ctx, doc.ObjectKey, 5*time.Minute)
}

func sanitizePropertyDocFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = propertyDocUnsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildPropertyDocObjectKey(workspaceID, propertyID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/patrimony/properties/%s/docs/%04d/%02d/%s-%s",
		workspaceID.String(), propertyID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
