package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SupplierCategory agrupa fornecedores na listagem.
type SupplierCategory string

const (
	SupplierCategoryServicosPublicos SupplierCategory = "servicos_publicos"
	SupplierCategoryTelecom          SupplierCategory = "telecom"
	SupplierCategoryStreaming         SupplierCategory = "streaming"
	SupplierCategoryVarejo           SupplierCategory = "varejo"
	SupplierCategoryFarmacia         SupplierCategory = "farmacia"
	SupplierCategorySaude            SupplierCategory = "saude"
	SupplierCategorySeguros          SupplierCategory = "seguros"
	SupplierCategoryFinanceiro       SupplierCategory = "financeiro"
	SupplierCategoryEducacao         SupplierCategory = "educacao"
	SupplierCategoryAlimentacao      SupplierCategory = "alimentacao"
	SupplierCategoryTransporte       SupplierCategory = "transporte"
	SupplierCategoryAcademia         SupplierCategory = "academia"
	SupplierCategoryMoradia          SupplierCategory = "moradia"
	SupplierCategoryTecnologia       SupplierCategory = "tecnologia"
	SupplierCategoryPet              SupplierCategory = "pet"
	SupplierCategoryJuridico         SupplierCategory = "juridico"
	SupplierCategoryContabil         SupplierCategory = "contabil"
	SupplierCategoryCondominio       SupplierCategory = "condominio"
	SupplierCategoryVestuario        SupplierCategory = "vestuario"
	SupplierCategoryBeleza           SupplierCategory = "beleza"
	SupplierCategoryViagem           SupplierCategory = "viagem"
	SupplierCategoryEntretenimento   SupplierCategory = "entretenimento"
	SupplierCategoryOutros           SupplierCategory = "outros"
)

// SupplierBillingType é o tipo de cobrança padrão do fornecedor.
type SupplierBillingType string

const (
	SupplierBillingBoleto          SupplierBillingType = "boleto"
	SupplierBillingPix             SupplierBillingType = "pix"
	SupplierBillingCartaoCredito   SupplierBillingType = "cartao_credito"
	SupplierBillingDebitoAutomtico SupplierBillingType = "debito_automatico"
	SupplierBillingDebito          SupplierBillingType = "debito"
	SupplierBillingTransferencia   SupplierBillingType = "transferencia"
	SupplierBillingDescontoFolha   SupplierBillingType = "desconto_folha"
)

// Supplier é um credor/payee que pode ser vinculado a despesas.
// WorkspaceID nil = fornecedor global (gerido pelo sistema, compartilhado).
// WorkspaceID não-nil = fornecedor criado pelo tenant.
type Supplier struct {
	ID                 uuid.UUID
	WorkspaceID        *uuid.UUID
	Name               string
	Category           SupplierCategory
	DefaultBillingType *SupplierBillingType
	PixKey             *string
	PixKeyHolder       *string
	BankName           *string
	BankAgency         *string
	BankAccount        *string
	BankAccountType    *string
	Notes              *string
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsGlobal indica que o fornecedor é gerido pelo sistema.
func (s *Supplier) IsGlobal() bool { return s.WorkspaceID == nil }

// Validate normaliza e valida o fornecedor.
func (s *Supplier) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return &ValidationError{Msg: "nome do fornecedor é obrigatório"}
	}
	if len(s.Name) > 150 {
		return &ValidationError{Msg: "nome do fornecedor excede o tamanho máximo"}
	}
	switch s.Category {
	case SupplierCategoryServicosPublicos, SupplierCategoryTelecom, SupplierCategoryStreaming,
		SupplierCategoryVarejo, SupplierCategoryFarmacia, SupplierCategorySaude,
		SupplierCategorySeguros, SupplierCategoryFinanceiro, SupplierCategoryEducacao,
		SupplierCategoryAlimentacao, SupplierCategoryTransporte, SupplierCategoryAcademia,
		SupplierCategoryMoradia, SupplierCategoryTecnologia, SupplierCategoryPet, SupplierCategoryJuridico,
		SupplierCategoryContabil, SupplierCategoryCondominio, SupplierCategoryVestuario,
		SupplierCategoryBeleza, SupplierCategoryViagem, SupplierCategoryEntretenimento,
		SupplierCategoryOutros:
	case "":
		s.Category = SupplierCategoryOutros
	default:
		return &ValidationError{Msg: "categoria do fornecedor inválida"}
	}
	if s.DefaultBillingType != nil {
		switch *s.DefaultBillingType {
		case SupplierBillingBoleto, SupplierBillingPix, SupplierBillingCartaoCredito,
			SupplierBillingDebitoAutomtico, SupplierBillingDebito, SupplierBillingTransferencia,
			SupplierBillingDescontoFolha:
		default:
			return &ValidationError{Msg: "tipo de cobrança inválido"}
		}
	}
	return nil
}

// SupplierFilter filtra a listagem de fornecedores.
type SupplierFilter struct {
	Query    string // busca por nome (case-insensitive)
	Category string
	Active   *bool
}

// SupplierRepository persiste fornecedores.
// List retorna globais + os do workspace, combinados.
type SupplierRepository interface {
	Create(ctx context.Context, s *Supplier) error
	GetByID(ctx context.Context, workspaceID uuid.UUID, id uuid.UUID) (*Supplier, error)
	Update(ctx context.Context, s *Supplier) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter SupplierFilter, limit, offset int) ([]Supplier, int64, error)
}
