package finance

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/storage"
)

// financeAllowedMimes são os tipos de arquivo aceitos para upload de documentos financeiros.
var financeAllowedMimes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// receiptAllowedMimes são os tipos aceitos para comprovantes de pagamento
// (mais permissivo: fotos de celular e documentos de texto).
var receiptAllowedMimes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/heic":         true,
	"image/heif":         true,
	"image/webp":         true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

var financeUnsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// FinanceDocumentService orquestra upload/download/listagem de documentos
// financeiros (faturas) sobre o object storage.
type FinanceDocumentService struct {
	repo           dom.FinanceDocumentRepository
	storage        storage.ObjectStorage
	maxUploadBytes int64
}

// NewFinanceDocumentService cria o serviço de documentos financeiros.
// maxUploadMB <= 0 usa o default (20MB).
func NewFinanceDocumentService(repo dom.FinanceDocumentRepository, st storage.ObjectStorage, maxUploadMB int) *FinanceDocumentService {
	if maxUploadMB <= 0 {
		maxUploadMB = 20
	}
	return &FinanceDocumentService{
		repo:           repo,
		storage:        st,
		maxUploadBytes: int64(maxUploadMB) * 1024 * 1024,
	}
}

type UploadFinanceDocInput struct {
	WorkspaceID      uuid.UUID
	UploadedByUserID uuid.UUID
	CardID           *uuid.UUID
	// Kind: papel do documento ('import' fatura, 'fiscal' cupom/nota). Default: import.
	Kind             dom.DocumentKind
	OriginalFileName string
	MimeType         string
	Size             int64
	Content          io.Reader
}

func (s *FinanceDocumentService) Upload(ctx context.Context, in UploadFinanceDocInput) (*dom.FinanceDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	if !financeAllowedMimes[strings.ToLower(strings.TrimSpace(in.MimeType))] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido (apenas PDF, JPEG ou PNG)"}
	}

	kind := in.Kind
	if kind == "" {
		kind = dom.DocumentImport
	}
	safeName := sanitizeFinanceFileName(in.OriginalFileName)
	objectKey := buildFinanceObjectKey(in.WorkspaceID, in.CardID, safeName)

	now := time.Now().UTC()
	doc := &dom.FinanceDocument{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		CardID:           in.CardID,
		Kind:             kind,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		MimeType:         strings.ToLower(strings.TrimSpace(in.MimeType)),
		SizeBytes:        in.Size,
		StorageProvider:  "minio",
		Bucket:           financeBucket,
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

// LoadContent retorna o documento e seu conteúdo bruto do storage (para extração).
func (s *FinanceDocumentService) LoadContent(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinanceDocument, []byte, error) {
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

func (s *FinanceDocumentService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinanceDocument, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListFinanceDocumentsResult struct {
	Items []dom.FinanceDocument
	Total int64
}

func (s *FinanceDocumentService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FinanceDocumentFilter, limit, offset int) (*ListFinanceDocumentsResult, error) {
	// A listagem geral mostra faturas importadas (default) ou cupons/notas
	// fiscais (kind=fiscal); comprovantes são listados pelo lançamento.
	if filter.Kind == nil || *filter.Kind == "" || *filter.Kind == dom.DocumentReceipt {
		kind := dom.DocumentImport
		filter.Kind = &kind
	}
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListFinanceDocumentsResult{Items: items, Total: total}, nil
}

// UploadReceipt anexa um comprovante de pagamento a um lançamento.
// A existência do lançamento no workspace deve ser validada pelo chamador.
func (s *FinanceDocumentService) UploadReceipt(ctx context.Context, in UploadFinanceDocInput, entryID uuid.UUID) (*dom.FinanceDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !receiptAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido para comprovante (PDF, imagem ou DOC)"}
	}

	kind := in.Kind
	if kind == "" {
		kind = dom.DocumentImport
	}
	safeName := sanitizeFinanceFileName(in.OriginalFileName)
	objectKey := buildReceiptObjectKey(in.WorkspaceID, entryID, safeName)

	now := time.Now().UTC()
	doc := &dom.FinanceDocument{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		EntryID:          &entryID,
		Kind:             dom.DocumentReceipt,
		FileName:         safeName,
		OriginalFileName: in.OriginalFileName,
		MimeType:         mime,
		SizeBytes:        in.Size,
		StorageProvider:  "minio",
		Bucket:           financeBucket,
		ObjectKey:        objectKey,
		UploadedByUserID: in.UploadedByUserID,
		ExtractionStatus: dom.ExtractionNotRequired,
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

// ListReceipts lista os comprovantes anexados a um lançamento.
func (s *FinanceDocumentService) ListReceipts(ctx context.Context, workspaceID, entryID uuid.UUID, limit, offset int) (*ListFinanceDocumentsResult, error) {
	kind := dom.DocumentReceipt
	items, total, err := s.repo.List(ctx, workspaceID, dom.FinanceDocumentFilter{Kind: &kind, EntryID: &entryID}, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListFinanceDocumentsResult{Items: items, Total: total}, nil
}

func (s *FinanceDocumentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// UpdateExtraction persiste extraction_status/extracted_text/extracted_json/entry_id
// do documento (usado, por ex., ao vincular a fatura criada na confirmação).
func (s *FinanceDocumentService) UpdateExtraction(ctx context.Context, doc *dom.FinanceDocument) error {
	return s.repo.UpdateExtraction(ctx, doc)
}

// DownloadURL gera uma URL presignada de download (validade 5 minutos).
func (s *FinanceDocumentService) DownloadURL(ctx context.Context, workspaceID, id uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return "", err
	}
	if !s.storage.Enabled() {
		return "", &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	return s.storage.PresignedGetURL(ctx, doc.ObjectKey, 5*time.Minute)
}

// financeBucket é o nome lógico do bucket persistido no registro.
// O bucket físico é resolvido pelo storage.
const financeBucket = "finance"

func sanitizeFinanceFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = financeUnsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._")
	if base == "" {
		return "arquivo"
	}
	if len(base) > 200 {
		base = base[len(base)-200:]
	}
	return base
}

func buildReceiptObjectKey(workspaceID, entryID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/finance/receipts/%s/%04d/%02d/%s-%s",
		workspaceID.String(), entryID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}

func buildFinanceObjectKey(workspaceID uuid.UUID, cardID *uuid.UUID, fileName string) string {
	card := "none"
	if cardID != nil && *cardID != uuid.Nil {
		card = cardID.String()
	}
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/finance/cards/%s/%04d/%02d/%s-%s",
		workspaceID.String(), card, now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
