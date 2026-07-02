package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MemberDocType classifica o documento pessoal do membro da família.
type MemberDocType string

const (
	MemberDocCPF              MemberDocType = "cpf"
	MemberDocRG               MemberDocType = "rg"
	MemberDocCNH              MemberDocType = "cnh"
	MemberDocPassaporte       MemberDocType = "passaporte"
	MemberDocCarteiraTrabalho MemberDocType = "carteira_trabalho"
	MemberDocCertidaoNasc     MemberDocType = "certidao_nascimento"
	MemberDocTituloEleitor    MemberDocType = "titulo_eleitor"
	MemberDocCartaoSUS        MemberDocType = "cartao_sus"
	MemberDocPlanoSaude       MemberDocType = "plano_saude"
	MemberDocOutro            MemberDocType = "outro"
)

// ValidMemberDocType informa se o tipo de documento é conhecido.
func ValidMemberDocType(t MemberDocType) bool {
	switch t {
	case MemberDocCPF, MemberDocRG, MemberDocCNH, MemberDocPassaporte,
		MemberDocCarteiraTrabalho, MemberDocCertidaoNasc, MemberDocTituloEleitor,
		MemberDocCartaoSUS, MemberDocPlanoSaude, MemberDocOutro:
		return true
	}
	return false
}

// MemberDocument é um documento pessoal (arquivo) de um membro da família.
// Mapeia a tabela family_member_documents.
type MemberDocument struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	FamilyMemberID   uuid.UUID
	DocType          MemberDocType
	Label            *string // rótulo livre (obrigatório quando DocType = outro)
	DocNumber        *string
	ValidUntil       *time.Time // cnh/passaporte vencem
	Notes            *string
	FileName         string
	OriginalFileName string
	MimeType         string
	SizeBytes        int64
	StorageProvider  string
	Bucket           string
	ObjectKey        string
	UploadedByUserID uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate valida invariantes do documento do membro.
func (d *MemberDocument) Validate() error {
	if d.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if d.FamilyMemberID == uuid.Nil {
		return &ValidationError{Msg: "family_member_id é obrigatório"}
	}
	if !ValidMemberDocType(d.DocType) {
		return &ValidationError{Msg: "doc_type inválido"}
	}
	if d.DocType == MemberDocOutro && (d.Label == nil || strings.TrimSpace(*d.Label) == "") {
		return &ValidationError{Msg: "label é obrigatório quando doc_type = outro"}
	}
	if d.UploadedByUserID == uuid.Nil {
		return &ValidationError{Msg: "uploaded_by_user_id é obrigatório"}
	}
	if strings.TrimSpace(d.FileName) == "" {
		return &ValidationError{Msg: "file_name é obrigatório"}
	}
	if strings.TrimSpace(d.ObjectKey) == "" {
		return &ValidationError{Msg: "object_key é obrigatório"}
	}
	if strings.TrimSpace(d.Bucket) == "" {
		return &ValidationError{Msg: "bucket é obrigatório"}
	}
	return nil
}

// MemberDocumentRepository persiste documentos de membros (workspace-scoped, soft-delete).
type MemberDocumentRepository interface {
	Create(ctx context.Context, d *MemberDocument) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*MemberDocument, error)
	ListByMember(ctx context.Context, workspaceID, familyMemberID uuid.UUID, limit, offset int) ([]MemberDocument, int64, error)
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
}
