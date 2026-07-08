package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// FiscalService vincula cupons/notas fiscais (documentos kind=fiscal) a
// lançamentos, gravando o detalhamento item a item. Itens são informação —
// o valor da despesa vive no lançamento; nada aqui altera saldos.
type FiscalService struct {
	items    dom.FiscalItemRepository
	entries  dom.FinancialEntryRepository
	docs     dom.FinanceDocumentRepository
	entrySvc *FinancialEntryService
}

func NewFiscalService(
	items dom.FiscalItemRepository,
	entries dom.FinancialEntryRepository,
	docs dom.FinanceDocumentRepository,
	entrySvc *FinancialEntryService,
) *FiscalService {
	return &FiscalService{items: items, entries: entries, docs: docs, entrySvc: entrySvc}
}

// FiscalConfirmItem é um item revisado pelo usuário (valores em centavos,
// quantidade em milésimos).
type FiscalConfirmItem struct {
	Description   string
	QuantityMilli int64
	UnitCents     int64
	AmountCents   int64
	Category      *string
}

// FiscalConfirmInput confirma o vínculo do cupom: a uma despesa existente
// (EntryID) ou criando uma despesa nova (NewEntry).
type FiscalConfirmInput struct {
	WorkspaceID uuid.UUID
	DocumentID  uuid.UUID
	EntryID     *uuid.UUID
	NewEntry    *CreateEntryInput
	Items       []FiscalConfirmItem
}

// Confirm grava os itens do cupom e liga documento ↔ lançamento.
// Reconfirmar o mesmo lançamento substitui o detalhamento anterior.
func (s *FiscalService) Confirm(ctx context.Context, in FiscalConfirmInput) (*dom.FinancialEntry, []dom.FiscalItem, error) {
	if len(in.Items) == 0 {
		return nil, nil, &dom.ValidationError{Msg: "o cupom precisa de ao menos um item"}
	}
	doc, err := s.docs.GetByID(ctx, in.WorkspaceID, in.DocumentID)
	if err != nil {
		return nil, nil, err
	}
	if doc.Kind != dom.DocumentFiscal {
		return nil, nil, &dom.ValidationError{Msg: "documento não é um cupom/nota fiscal"}
	}

	// Resolve a despesa que agrupa a compra: existente ou criada agora.
	var entry *dom.FinancialEntry
	switch {
	case in.EntryID != nil:
		entry, err = s.entries.GetByID(ctx, in.WorkspaceID, *in.EntryID)
		if err != nil {
			return nil, nil, err
		}
		if entry.Kind != dom.KindDebit {
			return nil, nil, &dom.ValidationError{Msg: "cupom fiscal só pode ser vinculado a uma despesa"}
		}
	case in.NewEntry != nil:
		ne := *in.NewEntry
		ne.WorkspaceID = in.WorkspaceID
		ne.Kind = string(dom.KindDebit)
		ne.Recurrence = string(dom.RecurrenceNone)
		ne.InstallmentsTotal = nil
		created, cerr := s.entrySvc.Create(ctx, ne)
		if cerr != nil {
			return nil, nil, cerr
		}
		entry = &created[0]
	default:
		return nil, nil, &dom.ValidationError{Msg: "informe entry_id ou os dados da nova despesa"}
	}

	// Substitui detalhamento anterior (reimportação do cupom).
	if err := s.items.DeleteByEntry(ctx, in.WorkspaceID, entry.ID); err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	batch := make([]*dom.FiscalItem, 0, len(in.Items))
	for _, it := range in.Items {
		item := &dom.FiscalItem{
			ID:            uuid.New(),
			WorkspaceID:   in.WorkspaceID,
			EntryID:       entry.ID,
			DocumentID:    doc.ID,
			Description:   it.Description,
			QuantityMilli: it.QuantityMilli,
			UnitCents:     it.UnitCents,
			AmountCents:   it.AmountCents,
			Category:      it.Category,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := item.Validate(); err != nil {
			return nil, nil, err
		}
		batch = append(batch, item)
	}
	if err := s.items.CreateBatch(ctx, batch); err != nil {
		return nil, nil, err
	}

	// Liga lançamento → documento e documento → lançamento.
	entry.FiscalDocumentID = &doc.ID
	entry.UpdatedAt = now
	if err := s.entries.Update(ctx, entry); err != nil {
		return nil, nil, err
	}
	doc.EntryID = &entry.ID
	doc.UpdatedAt = now
	_ = s.docs.UpdateExtraction(ctx, doc)

	out := make([]dom.FiscalItem, len(batch))
	for i := range batch {
		out[i] = *batch[i]
	}
	return entry, out, nil
}

// ListByEntry retorna o detalhamento fiscal de um lançamento.
func (s *FiscalService) ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]dom.FiscalItem, error) {
	return s.items.ListByEntry(ctx, workspaceID, entryID)
}
