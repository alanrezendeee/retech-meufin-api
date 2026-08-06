package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EntryEventType classifica os eventos do ciclo de vida de um lançamento.
type EntryEventType string

const (
	// EventConfirmed: liquidação (rápida ou com desconto/parcial). Isenção
	// total também passa por aqui — o desconto integral no lançamento conta
	// a história.
	EventConfirmed EntryEventType = "confirmed"
	// EventSettled: liquidação detalhada (forma de pagamento, conta/cartão).
	EventSettled EntryEventType = "settled"
	// EventReopened: pagamento desfeito (realizada → prevista).
	EventReopened EntryEventType = "reopened"
	// EventCancelled: cancelamento, com o motivo do catálogo.
	EventCancelled EntryEventType = "cancelled"
	// EventDueDateChanged: prorrogação/antecipação do vencimento de um
	// lançamento previsto — a trilha preserva o vencimento original.
	EventDueDateChanged EntryEventType = "due_date_changed"
)

// EntryEvent é um registro imutável de transição no ciclo de vida do
// lançamento: o que aconteceu, quando, por quem e os valores envolvidos.
//
// É o que o updated_at nunca conseguiu ser: updated_at é "último toque" e
// qualquer edição posterior o sobrescreve — um rename em massa contaminou a
// coluna inteira. Evento não se edita; a resposta para "quando isso foi pago
// de verdade?" passa a existir por construção.
type EntryEvent struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	EntryID     uuid.UUID
	Event       EntryEventType
	FromStatus  *Status
	ToStatus    *Status
	// Detalhes de liquidação (confirmed/settled).
	PaidAt          *time.Time
	PaidAmountCents *int64
	// Motivo (cancelled).
	CancelReason *string
	// Vencimentos (due_date_changed).
	OldDueDate *time.Time
	NewDueDate *time.Time
	// ActorUserID: quem executou a ação (nulo em rotinas automáticas).
	ActorUserID *uuid.UUID
	CreatedAt   time.Time
}

// EntryEventRepository persiste a trilha (append-only: sem update nem delete).
type EntryEventRepository interface {
	Create(ctx context.Context, ev *EntryEvent) error
	ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]EntryEvent, error)
}
