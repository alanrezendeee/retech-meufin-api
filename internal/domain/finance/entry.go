package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Kind diferencia receita (credit) de despesa (debit).
type Kind string

const (
	KindCredit Kind = "credit"
	KindDebit  Kind = "debit"
)

// Status representa o ciclo de vida de um lançamento.
type Status string

const (
	StatusPrevista  Status = "prevista"
	StatusRealizada Status = "realizada"
	StatusCancelada Status = "cancelada"
)

// Recurrence define a periodicidade de geração das ocorrências.
type Recurrence string

const (
	RecurrenceNone    Recurrence = "none"
	RecurrenceWeekly  Recurrence = "weekly"
	RecurrenceMonthly Recurrence = "monthly"
	RecurrenceYearly  Recurrence = "yearly"
)

// PaymentMethod é a forma de pagamento (ou recebimento) usada na liquidação.
type PaymentMethod string

const (
	PaymentPix           PaymentMethod = "pix"
	PaymentDebito        PaymentMethod = "debito"
	PaymentTransferencia PaymentMethod = "transferencia"
	PaymentBoleto        PaymentMethod = "boleto"
	PaymentDinheiro      PaymentMethod = "dinheiro"
	PaymentCartaoCredito PaymentMethod = "cartao_credito"
)

// ValidPaymentMethod informa se o método de pagamento é conhecido.
func ValidPaymentMethod(m PaymentMethod) bool {
	switch m {
	case PaymentPix, PaymentDebito, PaymentTransferencia, PaymentBoleto, PaymentDinheiro, PaymentCartaoCredito:
		return true
	}
	return false
}

// FinancialEntry é um lançamento único de crédito ou débito.
type FinancialEntry struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	Kind              Kind
	Status            Status
	AmountCents       int64
	DueDate           time.Time
	FamilyMemberID    *uuid.UUID
	SourceID          *uuid.UUID
	Type              *string
	Description       string
	Recurrence        Recurrence
	RecurrenceGroupID *uuid.UUID
	CardID            *uuid.UUID
	ParentID          *uuid.UUID
	InstallmentNumber *int
	InstallmentTotal  *int
	Notes             *string
	// Liquidação: preenchidos quando o lançamento é pago/recebido.
	// PaidAmountCents pode diferir de AmountCents (juros/multa/desconto).
	PaidAt           *time.Time
	PaidAmountCents  *int64
	PaymentMethod    *PaymentMethod
	PaymentAccountID *uuid.UUID
	PaymentCardID    *uuid.UUID
	// Desconto obtido na liquidação; o motivo (slug do catálogo global
	// DiscountReasons) vira indicador para insights.
	DiscountCents  *int64
	DiscountReason *string
	// ResidualOfID aponta para o lançamento de origem quando este lançamento
	// nasceu de um pagamento parcial (desdobramento do saldo não pago).
	ResidualOfID *uuid.UUID
	// PurchaseDate é a data em que a compra foi realizada (itens de fatura).
	// Informacional: o vencimento do item é sempre o vencimento da fatura.
	PurchaseDate *time.Time
	// FiscalDocumentID aponta para o cupom/nota fiscal (finance_documents)
	// cujo detalhamento item a item está vinculado a este lançamento.
	FiscalDocumentID *uuid.UUID
	SupplierID       *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate valida invariantes do lançamento.
func (e *FinancialEntry) Validate() error {
	if e.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	switch e.Kind {
	case KindCredit, KindDebit:
	default:
		return &ValidationError{Msg: "kind do lançamento inválido"}
	}
	switch e.Status {
	case StatusPrevista, StatusRealizada, StatusCancelada:
	case "":
		e.Status = StatusPrevista
	default:
		return &ValidationError{Msg: "status do lançamento inválido"}
	}
	switch e.Recurrence {
	case RecurrenceNone, RecurrenceWeekly, RecurrenceMonthly, RecurrenceYearly:
	case "":
		e.Recurrence = RecurrenceNone
	default:
		return &ValidationError{Msg: "recurrence do lançamento inválido"}
	}
	if e.AmountCents == 0 {
		return &ValidationError{Msg: "amount_cents não pode ser zero"}
	}
	if e.DueDate.IsZero() {
		return &ValidationError{Msg: "due_date é obrigatória"}
	}
	if !validEntryType(e.Kind, e.Type) {
		if e.Kind == KindCredit {
			return &ValidationError{Msg: "type de receita fora do catálogo"}
		}
		return &ValidationError{Msg: "categoria de despesa fora do catálogo"}
	}
	if e.DiscountCents != nil {
		if *e.DiscountCents <= 0 {
			return &ValidationError{Msg: "discount_cents deve ser maior que zero"}
		}
		if *e.DiscountCents >= e.AmountCents {
			return &ValidationError{Msg: "desconto não pode ser maior ou igual ao valor do lançamento"}
		}
		if e.DiscountReason == nil || *e.DiscountReason == "" {
			return &ValidationError{Msg: "informe o motivo do desconto"}
		}
	}
	if e.DiscountReason != nil && *e.DiscountReason != "" {
		if !ValidDiscountReason(*e.DiscountReason) {
			return &ValidationError{Msg: "motivo de desconto fora do catálogo"}
		}
		if e.DiscountCents == nil {
			return &ValidationError{Msg: "motivo de desconto exige discount_cents"}
		}
	}
	e.Description = strings.TrimSpace(e.Description)
	return nil
}

// FinancialEntryFilter filtra a listagem de lançamentos.
type FinancialEntryFilter struct {
	Query          string // busca na descrição (case-insensitive)
	Kind           *string
	Status         *string
	FamilyMemberID *uuid.UUID
	Type           *string
	Year           *int
	Month          *int
	CardID         *uuid.UUID
	ParentID       *uuid.UUID
	TopLevelOnly   bool
	// Contas do dia: recortes por vencimento.
	DueOn      *time.Time // due_date == dia (ignora hora)
	DueFrom    *time.Time // due_date >= dia
	DueTo      *time.Time // due_date <= dia
	Overdue    bool       // due_date < hoje AND status = prevista
	SupplierID *uuid.UUID // filtra pelo fornecedor vinculado
}

// FinancialEntryRepository persiste lançamentos com escopo de workspace.
type FinancialEntryRepository interface {
	Create(ctx context.Context, e *FinancialEntry) error
	CreateBatch(ctx context.Context, es []*FinancialEntry) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FinancialEntry, error)
	Update(ctx context.Context, e *FinancialEntry) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter FinancialEntryFilter, limit, offset int) ([]FinancialEntry, int64, error)
	// CascadeStatusToChildren propaga o status da fatura pai para os filhos não cancelados
	// (liquidar/cancelar a fatura liquida/cancela os itens juntos). paidAt só é aplicado
	// quando status = realizada.
	CascadeStatusToChildren(ctx context.Context, workspaceID, parentID uuid.UUID, status Status, paidAt *time.Time) error
	// ListRecurrenceFrontiers retorna, para cada grupo de recorrência (todas as
	// workspaces), a ocorrência mais recente — o ponto de onde o extensor
	// completa o horizonte rolling.
	ListRecurrenceFrontiers(ctx context.Context) ([]FinancialEntry, error)
	// ListResiduals retorna os lançamentos residuais gerados a partir do
	// lançamento de origem (pagamento parcial).
	ListResiduals(ctx context.Context, workspaceID, originID uuid.UUID) ([]FinancialEntry, error)
	// ListInvoiceInstallments retorna compras parceladas dentro de faturas
	// (filhos com installment_number/total) — insumo da projeção.
	ListInvoiceInstallments(ctx context.Context, workspaceID uuid.UUID) ([]FinancialEntry, error)
	// ListFutureGroupSiblings retorna as ocorrências 'prevista' do grupo de
	// recorrência com due_date posterior a `after`, excluindo excludeID —
	// alvo da edição em série ("aplicar às próximas").
	ListFutureGroupSiblings(ctx context.Context, workspaceID, groupID uuid.UUID, after time.Time, excludeID uuid.UUID) ([]FinancialEntry, error)
}
