package finance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// Anexos de apoio ao lançamento (boleto, QR Code Pix, nota, contrato…).
// Diferem do comprovante (kind=receipt): não comprovam pagamento e carregam
// um "código de pagamento" opcional (linha digitável / Pix copia e cola) que
// o usuário copia na hora de pagar. Imagens com QR Code Pix têm o payload
// lido no servidor quando o usuário não informa o código.

// SetQRDecoder injeta o leitor de QR Code server-side (opcional; nil = sem
// leitura automática do Pix copia e cola em imagens).
func (s *FinanceDocumentService) SetQRDecoder(qr qrDecoder) { s.qr = qr }

// UploadAttachmentInput é a entrada do upload de anexo de lançamento.
type UploadAttachmentInput struct {
	WorkspaceID      uuid.UUID
	UploadedByUserID uuid.UUID
	EntryID          uuid.UUID
	Type             dom.AttachmentType
	// PaymentCode informado pelo usuário (linha digitável ou Pix copia e cola).
	// Vazio = tenta ler do QR Code quando a imagem tiver um.
	PaymentCode      string
	Note             string
	OriginalFileName string
	MimeType         string
	Size             int64
	Content          io.Reader
}

// UploadAttachment anexa um documento de apoio ao lançamento. A existência do
// lançamento no workspace deve ser validada pelo chamador.
func (s *FinanceDocumentService) UploadAttachment(ctx context.Context, in UploadAttachmentInput) (*dom.FinanceDocument, error) {
	if !s.storage.Enabled() {
		return nil, &dom.ValidationError{Msg: "armazenamento de documentos indisponível (storage não configurado)"}
	}
	if in.EntryID == uuid.Nil {
		return nil, &dom.ValidationError{Msg: "entry_id é obrigatório"}
	}
	if !dom.ValidAttachmentType(in.Type) {
		return nil, &dom.ValidationError{Msg: "attachment_type inválido (boleto, pix_qrcode, nota_fiscal, contrato, fatura ou outro)"}
	}
	if in.Size <= 0 {
		return nil, &dom.ValidationError{Msg: "arquivo vazio"}
	}
	if in.Size > s.maxUploadBytes {
		return nil, &dom.ValidationError{Msg: fmt.Sprintf("arquivo excede o limite de %d MB", s.maxUploadBytes/(1024*1024))}
	}
	mime := strings.ToLower(strings.TrimSpace(in.MimeType))
	if !receiptAllowedMimes[mime] {
		return nil, &dom.ValidationError{Msg: "tipo de arquivo não permitido para anexo (PDF, imagem ou DOC)"}
	}

	meta := dom.AttachmentMeta{Type: in.Type}
	if note := strings.TrimSpace(in.Note); note != "" {
		if len(note) > 500 {
			return nil, &dom.ValidationError{Msg: "note excede 500 caracteres"}
		}
		meta.Note = &note
	}

	// Código de pagamento: o digitado vale mais que o lido da imagem.
	code, err := NormalizePaymentCode(in.Type, in.PaymentCode)
	if err != nil {
		return nil, err
	}
	content := in.Content
	if code == "" && s.qr != nil && strings.HasPrefix(mime, "image/") &&
		(in.Type == dom.AttachmentPixQRCode || in.Type == dom.AttachmentBoleto) {
		// Precisamos dos bytes duas vezes (decode + upload); o limite de tamanho
		// já foi validado, então bufferizar é seguro.
		buf, err := io.ReadAll(io.LimitReader(in.Content, in.Size))
		if err != nil {
			return nil, fmt.Errorf("falha ao ler arquivo: %w", err)
		}
		content = bytes.NewReader(buf)
		if payload, ok := s.qr.DecodeNFCe(buf); ok && IsPixPayload(payload) {
			code = strings.TrimSpace(payload)
			src := "qrcode"
			meta.PaymentCodeSource = &src
		}
	}
	if code != "" {
		meta.PaymentCode = &code
		if meta.PaymentCodeSource == nil {
			src := "user"
			meta.PaymentCodeSource = &src
		}
	}

	safeName := sanitizeFinanceFileName(in.OriginalFileName)
	objectKey := buildAttachmentObjectKey(in.WorkspaceID, in.EntryID, safeName)

	now := time.Now().UTC()
	entryID := in.EntryID
	doc := &dom.FinanceDocument{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		EntryID:          &entryID,
		Kind:             dom.DocumentAttachment,
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
	if err := doc.SetAttachmentMeta(meta); err != nil {
		return nil, err
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}

	if err := s.storage.Put(ctx, objectKey, content, in.Size, doc.MimeType); err != nil {
		return nil, fmt.Errorf("falha ao enviar arquivo: %w", err)
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// ListAttachments lista os anexos de apoio de um lançamento.
func (s *FinanceDocumentService) ListAttachments(ctx context.Context, workspaceID, entryID uuid.UUID, limit, offset int) (*ListFinanceDocumentsResult, error) {
	kind := dom.DocumentAttachment
	items, total, err := s.repo.List(ctx, workspaceID, dom.FinanceDocumentFilter{Kind: &kind, EntryID: &entryID}, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListFinanceDocumentsResult{Items: items, Total: total}, nil
}

var nonDigits = regexp.MustCompile(`\D`)

// NormalizePaymentCode limpa e valida o código de pagamento conforme o tipo
// do anexo. Vazio é permitido (anexo sem código). Para boleto aceita a linha
// digitável (47 dígitos, bancário) ou a de arrecadação (48), com ou sem
// pontuação; para Pix exige o payload EMV ("000201…"). Outros tipos aceitam
// texto livre curto.
func NormalizePaymentCode(t dom.AttachmentType, raw string) (string, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		return "", nil
	}
	switch t {
	case dom.AttachmentBoleto:
		digits := nonDigits.ReplaceAllString(code, "")
		if len(digits) != 47 && len(digits) != 48 {
			return "", &dom.ValidationError{Msg: "linha digitável do boleto deve ter 47 ou 48 dígitos"}
		}
		return digits, nil
	case dom.AttachmentPixQRCode:
		if !IsPixPayload(code) {
			return "", &dom.ValidationError{Msg: "código Pix inválido (esperado o 'copia e cola' iniciando em 000201)"}
		}
		return code, nil
	default:
		if len(code) > 512 {
			return "", &dom.ValidationError{Msg: "payment_code excede 512 caracteres"}
		}
		return code, nil
	}
}

// IsPixPayload reconhece um payload EMV de Pix (BR Code): começa com o
// Payload Format Indicator "000201" e contém o GUI "br.gov.bcb.pix".
func IsPixPayload(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "000201") && strings.Contains(strings.ToLower(s), "br.gov.bcb.pix")
}

func buildAttachmentObjectKey(workspaceID, entryID uuid.UUID, fileName string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("tenants/%s/finance/attachments/%s/%04d/%02d/%s-%s",
		workspaceID.String(), entryID.String(), now.Year(), int(now.Month()), uuid.New().String(), fileName)
}
