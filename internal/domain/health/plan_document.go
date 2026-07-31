package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PlanDocType classifica um documento anexado a um plano de saúde.
type PlanDocType string

const (
	PlanDocContrato            PlanDocType = "contrato"       // contrato/apólice
	PlanDocCarteirinha         PlanDocType = "carteirinha"    // cartão do plano
	PlanDocManual              PlanDocType = "manual"         // manual/condições gerais
	PlanDocTabelaCoparticipacao PlanDocType = "tabela_coparticipacao"
	PlanDocAditivoReajuste     PlanDocType = "aditivo_reajuste"
	PlanDocBoleto              PlanDocType = "boleto"         // comprovante de mensalidade
	PlanDocTermoAdesao         PlanDocType = "termo_adesao"
	PlanDocComprovanteCarencia PlanDocType = "comprovante_carencia"
	PlanDocFormularioReembolso PlanDocType = "formulario_reembolso"
	PlanDocDeclaracaoIR        PlanDocType = "declaracao_ir" // quitação anual p/ IR
	PlanDocRedeCredenciada     PlanDocType = "rede_credenciada"
	PlanDocLaudo               PlanDocType = "laudo"          // laudo p/ autorização
	PlanDocOutro               PlanDocType = "outro"
)

// ValidPlanDocType informa se o tipo é conhecido.
func ValidPlanDocType(t PlanDocType) bool {
	switch t {
	case PlanDocContrato, PlanDocCarteirinha, PlanDocManual, PlanDocTabelaCoparticipacao,
		PlanDocAditivoReajuste, PlanDocBoleto, PlanDocTermoAdesao, PlanDocComprovanteCarencia,
		PlanDocFormularioReembolso, PlanDocDeclaracaoIR, PlanDocRedeCredenciada, PlanDocLaudo,
		PlanDocOutro:
		return true
	}
	return false
}

// PlanDocument é um documento (arquivo) anexado a um plano de saúde.
// Mapeia a tabela health_plan_documents.
type PlanDocument struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	PlanID           uuid.UUID
	DocType          PlanDocType
	Label            *string    // rótulo livre (obrigatório quando DocType = outro)
	DocNumber        *string    // ex.: nº da apólice/carteirinha
	ValidUntil       *time.Time // validade quando aplicável
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

// Validate valida invariantes do documento do plano.
func (d *PlanDocument) Validate() error {
	if d.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if d.PlanID == uuid.Nil {
		return &ValidationError{Msg: "plan_id é obrigatório"}
	}
	if !ValidPlanDocType(d.DocType) {
		return &ValidationError{Msg: "doc_type inválido"}
	}
	if d.DocType == PlanDocOutro && (d.Label == nil || strings.TrimSpace(*d.Label) == "") {
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

// PlanDocumentRepository persiste documentos de planos (workspace-scoped, soft-delete).
type PlanDocumentRepository interface {
	Create(ctx context.Context, d *PlanDocument) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*PlanDocument, error)
	ListByPlan(ctx context.Context, workspaceID, planID uuid.UUID, limit, offset int) ([]PlanDocument, int64, error)
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
}
