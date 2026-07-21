package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FiscalItem é um item de cupom/nota fiscal — detalhamento informacional de
// um lançamento (o valor da despesa já está no lançamento pai; itens NÃO são
// obrigações financeiras e não entram em agregações de saldo).
type FiscalItem struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	// EntryID é o lançamento que o cupom detalha (avulso ou compra de fatura).
	EntryID uuid.UUID
	// DocumentID é o cupom/nota fiscal de origem (finance_documents, kind=fiscal).
	DocumentID  uuid.UUID
	Description string
	// QuantityMilli é a quantidade em milésimos (3 casas): 1un = 1000,
	// 0,455kg = 455 — inteiro sempre, mesma filosofia dos centavos.
	QuantityMilli int64
	UnitCents     int64
	AmountCents   int64
	Category      *string
	// UnitOfMeasure é a unidade de medida (kg, un, L, g…), normalizada em
	// maiúsculas. Base para o preço por unidade canônica no painel. Nil quando
	// desconhecida (leitura por IA sem unidade, ou dados anteriores).
	UnitOfMeasure *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validate valida invariantes do item fiscal.
func (i *FiscalItem) Validate() error {
	if i.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if i.EntryID == uuid.Nil {
		return &ValidationError{Msg: "entry_id é obrigatório"}
	}
	if i.DocumentID == uuid.Nil {
		return &ValidationError{Msg: "document_id é obrigatório"}
	}
	i.Description = strings.TrimSpace(i.Description)
	if i.Description == "" {
		return &ValidationError{Msg: "descrição do item é obrigatória"}
	}
	if i.AmountCents <= 0 {
		return &ValidationError{Msg: "amount_cents do item deve ser maior que zero"}
	}
	if i.QuantityMilli < 0 || i.UnitCents < 0 {
		return &ValidationError{Msg: "quantidade e valor unitário não podem ser negativos"}
	}
	return nil
}

// FiscalItemRepository persiste itens de cupom/nota fiscal.
type FiscalItemRepository interface {
	CreateBatch(ctx context.Context, items []*FiscalItem) error
	ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]FiscalItem, error)
	DeleteByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) error
	// ReassignEntry move os itens de um lançamento para outro (conciliação:
	// mover o detalhamento do cupom para a compra da fatura).
	ReassignEntry(ctx context.Context, workspaceID, fromEntryID, toEntryID uuid.UUID) error
}
