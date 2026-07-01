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

// allowedMimes são os tipos de arquivo aceitos para upload de documentos de saúde.
var allowedMimes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

var unsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type DocumentService struct {
	repo           dom.DocumentRepository
	storage        storage.ObjectStorage
	maxUploadBytes int64
}

// NewDocumentService cria o serviço de documentos. maxUploadMB <= 0 usa o default (20MB).
func NewDocumentService(repo dom.DocumentRepository, st storage.ObjectStorage, maxUploadMB int) *DocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &DocumentService{
		repo:           repo,
		storage:        st,
		maxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
}

// LoadContent retorna o documento e seu conteúdo bruto do storage (para extração).
func (s *DocumentService) LoadContent(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Document, []byte, error) {
	doc, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, nil, err
	}
	if !s.storage.Enabled() {
		return nil, nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	rc, err := s.storage.Get(ctx, doc.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, nil, err
	}
	return doc, data, nil
}

type UploadDocumentInput struct {
	WorkspaceID      uuid.UUID
	UploadedByUserID uuid.UUID
	DocumentType     string
	FamilyMemberID   *uuid.UUID
	LabID            *uuid.UUID
	ExamRequestID    *uuid.UUID
	ExamResultID     *uuid.UUID
	OriginalFileName string
	MimeType         string
	Size             int64
	Content          io.Reader
}

func (s *DocumentService) Upload(ctx context.Context, in UploadDocumentInput) (*dom.Document, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	if !allowedMimes[strings.ToLower(strings.TrimSpace(in.MimeType))] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (apenas PDF, JPEG ou PNG)"}
	}
	docType := dom.DocumentType(strings.TrimSpace(in.DocumentType))
	if !dom.ValidDocumentType(docType) {
		return nil, &dom.ValidationError{Msg: "document_type inválido"}
	}

	safeName := sanitizeFileName(in.OriginalFileName)
	objectKey := buildObjectKey(in.WorkspaceID, in.FamilyMemberID, safeName)
	bucket := storageBucketFromKey() // bucket é gerenciado pelo storage; guardamos o nome lógico

	now := time.Now().UTC()
	doc := &dom.Document{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		FamilyMemberID:   in.FamilyMemberID,
		LabID:            in.LabID,
		ExamRequestID:    in.ExamRequestID,
		ExamResultID:     in.ExamResultID,
		DocumentType:     docType,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		MimeType:         strings.ToLower(strings.TrimSpace(in.MimeType)),
		SizeBytes:        in.Size,
		StorageProvider:  "minio",
		Bucket:           bucket,
		ObjectKey:        objectKey,
		UploadedByUserID: in.UploadedByUserID,
		ExtractionStatus: dom.ExtractionPending,
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

func (s *DocumentService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Document, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListDocumentsResult struct {
	Items []dom.Document
	Total int64
}

func (s *DocumentService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListDocumentsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListDocumentsResult{Items: items, Total: total}, nil
}

func (s *DocumentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
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

// storageBucketFromKey retorna o nome lógico do bucket persistido no registro.
// O bucket físico é resolvido pelo storage; aqui gravamos o valor de referência.
func storageBucketFromKey() string { return "health" }

func sanitizeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = unsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildObjectKey(workspaceID uuid.UUID, familyMemberID *uuid.UUID, fileName string) string {
	member := "none"
	if familyMemberID != nil && *familyMemberID != uuid.Nil {
		member = familyMemberID.String()
	}
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/health/%s/%04d/%02d/%s-%s",
		workspaceID.String(), member, now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
