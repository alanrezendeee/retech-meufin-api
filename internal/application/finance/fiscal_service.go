package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// FiscalService vincula cupons/notas fiscais (documentos kind=fiscal) a
// lançamentos, gravando o detalhamento item a item. Itens são informação —
// o valor da despesa vive no lançamento; nada aqui altera saldos.
type FiscalService struct {
	items      dom.FiscalItemRepository
	entries    dom.FinancialEntryRepository
	docs       dom.FinanceDocumentRepository
	entrySvc   *FinancialEntryService
	categories *ExpenseCategoryService // auto-cadastro de categorias novas (opcional)
}

func NewFiscalService(
	items dom.FiscalItemRepository,
	entries dom.FinancialEntryRepository,
	docs dom.FinanceDocumentRepository,
	entrySvc *FinancialEntryService,
	categories *ExpenseCategoryService,
) *FiscalService {
	return &FiscalService{items: items, entries: entries, docs: docs, entrySvc: entrySvc, categories: categories}
}

// FiscalConfirmItem é um item revisado pelo usuário (valores em centavos,
// quantidade em milésimos). CategoryName/CategoryGroup acompanham categorias
// NOVAS (a criar) no auto-cadastro.
type FiscalConfirmItem struct {
	Description   string
	QuantityMilli int64
	UnitCents     int64
	AmountCents   int64
	Category      *string
	CategoryName  *string
	CategoryGroup *string
	// Unit é a unidade de medida (kg, un, L…); normalizada e persistida.
	Unit *string
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
// Categorias novas mantidas pelo usuário são auto-cadastradas (dedup + teto);
// as criadas voltam para a UI informar.
func (s *FiscalService) Confirm(ctx context.Context, in FiscalConfirmInput) (*dom.FinancialEntry, []dom.FiscalItem, []dom.ExpenseCategory, error) {
	if len(in.Items) == 0 {
		return nil, nil, nil, &dom.ValidationError{Msg: "o cupom precisa de ao menos um item"}
	}
	doc, err := s.docs.GetByID(ctx, in.WorkspaceID, in.DocumentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if doc.Kind != dom.DocumentFiscal {
		return nil, nil, nil, &dom.ValidationError{Msg: "documento não é um cupom/nota fiscal"}
	}

	// Resolve/auto-cadastra as categorias dos itens (dedup + teto por cupom).
	items, createdCats, err := s.ensureItemCategories(ctx, in.WorkspaceID, in.Items)
	if err != nil {
		return nil, nil, nil, err
	}

	// Resolve a despesa que agrupa a compra: existente ou criada agora.
	var entry *dom.FinancialEntry
	switch {
	case in.EntryID != nil:
		entry, err = s.entries.GetByID(ctx, in.WorkspaceID, *in.EntryID)
		if err != nil {
			return nil, nil, nil, err
		}
		if entry.Kind != dom.KindDebit {
			return nil, nil, nil, &dom.ValidationError{Msg: "cupom fiscal só pode ser vinculado a uma despesa"}
		}
	case in.NewEntry != nil:
		ne := *in.NewEntry
		ne.WorkspaceID = in.WorkspaceID
		ne.Kind = string(dom.KindDebit)
		ne.Recurrence = string(dom.RecurrenceNone)
		ne.InstallmentsTotal = nil
		created, cerr := s.entrySvc.Create(ctx, ne)
		if cerr != nil {
			return nil, nil, nil, cerr
		}
		entry = &created[0]
	default:
		return nil, nil, nil, &dom.ValidationError{Msg: "informe entry_id ou os dados da nova despesa"}
	}

	// Substitui detalhamento anterior (reimportação do cupom).
	if err := s.items.DeleteByEntry(ctx, in.WorkspaceID, entry.ID); err != nil {
		return nil, nil, nil, err
	}

	now := time.Now().UTC()
	batch := make([]*dom.FiscalItem, 0, len(items))
	for _, it := range items {
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
			UnitOfMeasure: normalizeUnitPtr(it.Unit),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := item.Validate(); err != nil {
			return nil, nil, nil, err
		}
		batch = append(batch, item)
	}
	if err := s.items.CreateBatch(ctx, batch); err != nil {
		return nil, nil, nil, err
	}

	// Liga lançamento → documento e documento → lançamento.
	entry.FiscalDocumentID = &doc.ID
	entry.UpdatedAt = now
	if err := s.entries.Update(ctx, entry); err != nil {
		return nil, nil, nil, err
	}
	doc.EntryID = &entry.ID
	doc.UpdatedAt = now
	_ = s.docs.UpdateExtraction(ctx, doc)

	out := make([]dom.FiscalItem, len(batch))
	for i := range batch {
		out[i] = *batch[i]
	}
	return entry, out, createdCats, nil
}

// ensureItemCategories resolve a categoria de cada item: mantém a que já existe
// na tenant; cria a nova (com grupo global válido, respeitando o teto por cupom)
// e devolve as criadas; o que não puder criar cai em "outros". Controle rígido
// anti-duplicação: reuso por slug já existente + idempotência do EnsureBySlug.
func (s *FiscalService) ensureItemCategories(ctx context.Context, workspaceID uuid.UUID, items []FiscalConfirmItem) ([]FiscalConfirmItem, []dom.ExpenseCategory, error) {
	out := make([]FiscalConfirmItem, len(items))
	copy(out, items)
	if s.categories == nil {
		return out, nil, nil
	}

	cats, err := s.categories.List(ctx, workspaceID)
	if err != nil {
		return nil, nil, err
	}
	active := make(map[string]struct{}, len(cats))
	for _, c := range cats {
		if c.Active {
			active[c.Slug] = struct{}{}
		}
	}

	var created []dom.ExpenseCategory
	newCount := 0
	for i := range out {
		it := &out[i]
		if it.Category == nil {
			continue
		}
		slug := strings.TrimSpace(*it.Category)
		if slug == "" {
			it.Category = nil
			continue
		}
		if _, ok := active[slug]; ok {
			continue // já existe: nada a criar
		}
		// Não existe: tenta criar se tem grupo global válido e está sob o teto.
		group := ""
		if it.CategoryGroup != nil {
			group = strings.TrimSpace(*it.CategoryGroup)
		}
		name := slug
		if it.CategoryName != nil && strings.TrimSpace(*it.CategoryName) != "" {
			name = strings.TrimSpace(*it.CategoryName)
		}
		if _, gok := dom.ExpenseGroups[group]; gok && newCount < MaxNewFiscalCategoriesPerReceipt {
			cat, didCreate, cerr := s.categories.EnsureBySlug(ctx, workspaceID, slug, name, group)
			if cerr == nil {
				active[slug] = struct{}{}
				if didCreate {
					created = append(created, *cat)
					newCount++
				}
				continue
			}
		}
		// Fallback: sem grupo válido, acima do teto ou erro → "outros".
		fb := dom.FallbackCategorySlug
		it.Category = &fb
	}
	return out, created, nil
}

// ListByEntry retorna o detalhamento fiscal de um lançamento.
func (s *FiscalService) ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]dom.FiscalItem, error) {
	return s.items.ListByEntry(ctx, workspaceID, entryID)
}
