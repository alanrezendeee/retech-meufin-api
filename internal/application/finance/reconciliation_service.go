package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// reconcileDateWindowDays é a tolerância de data entre a compra do cupom e a
// linha da fatura (a data de lançamento na fatura pode diferir da compra).
const reconcileDateWindowDays = 4

// InvoicePurchase é uma compra dentro de uma fatura (candidata a receber o
// detalhamento de um cupom).
type InvoicePurchase struct {
	EntryID     uuid.UUID
	AmountCents int64
	Date        time.Time
	Description string
}

// ReconcileCandidate é um cupom pago no CRÉDITO já lançado como despesa avulsa
// (fora de fatura) — candidato a ser conciliado com uma compra da fatura.
type ReconcileCandidate struct {
	CupomEntryID uuid.UUID
	DocumentID   uuid.UUID
	AmountCents  int64
	Date         time.Time
	Merchant     string
}

// ReconcileMatch é uma sugestão de conciliação (compra da fatura ↔ cupom).
// NUNCA aplicada automaticamente: o usuário confirma cada uma.
type ReconcileMatch struct {
	Purchase InvoicePurchase
	Cupom    ReconcileCandidate
	DaysDiff int
}

// ReconciliationRepository consulta compras de fatura e cupons conciliáveis.
type ReconciliationRepository interface {
	InvoicePurchases(ctx context.Context, workspaceID, invoiceEntryID uuid.UUID) ([]InvoicePurchase, error)
	ReconcilableCupons(ctx context.Context, workspaceID uuid.UUID) ([]ReconcileCandidate, error)
}

// ReconciliationService casa cupons de cartão com compras da fatura e aplica a
// conciliação (mover itens + vincular documento + remover a despesa avulsa).
type ReconciliationService struct {
	repo    ReconciliationRepository
	entries dom.FinancialEntryRepository
	items   dom.FiscalItemRepository
	docs    dom.FinanceDocumentRepository
}

func NewReconciliationService(
	repo ReconciliationRepository,
	entries dom.FinancialEntryRepository,
	items dom.FiscalItemRepository,
	docs dom.FinanceDocumentRepository,
) *ReconciliationService {
	return &ReconciliationService{repo: repo, entries: entries, items: items, docs: docs}
}

// SuggestForInvoice devolve as conciliações SUGERIDAS para uma fatura: casa cada
// compra com um cupom de crédito de mesmo valor e data próxima. Cada cupom é
// usado no máximo uma vez. Não altera nada — é só sugestão.
func (s *ReconciliationService) SuggestForInvoice(ctx context.Context, workspaceID, invoiceEntryID uuid.UUID) ([]ReconcileMatch, error) {
	purchases, err := s.repo.InvoicePurchases(ctx, workspaceID, invoiceEntryID)
	if err != nil {
		return nil, err
	}
	cupons, err := s.repo.ReconcilableCupons(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	used := make(map[uuid.UUID]bool, len(cupons))
	out := make([]ReconcileMatch, 0)
	for _, p := range purchases {
		bestIdx := -1
		bestDiff := reconcileDateWindowDays + 1
		for i := range cupons {
			c := cupons[i]
			if used[c.CupomEntryID] || c.AmountCents != p.AmountCents {
				continue
			}
			diff := absInt(daysBetween(c.Date, p.Date))
			if diff <= reconcileDateWindowDays && diff < bestDiff {
				bestIdx = i
				bestDiff = diff
			}
		}
		if bestIdx >= 0 {
			used[cupons[bestIdx].CupomEntryID] = true
			out = append(out, ReconcileMatch{Purchase: p, Cupom: cupons[bestIdx], DaysDiff: bestDiff})
		}
	}
	return out, nil
}

// Reconcile aplica a conciliação escolhida pelo usuário: move o detalhamento do
// cupom para a compra da fatura, vincula o documento à compra e REMOVE a despesa
// avulsa do cupom (elimina o duplo lançamento). Retorna a compra atualizada.
func (s *ReconciliationService) Reconcile(ctx context.Context, workspaceID, cupomEntryID, targetEntryID uuid.UUID) (*dom.FinancialEntry, error) {
	if cupomEntryID == targetEntryID {
		return nil, &dom.ValidationError{Msg: "cupom e compra de destino não podem ser o mesmo lançamento"}
	}
	cupom, err := s.entries.GetByID(ctx, workspaceID, cupomEntryID)
	if err != nil {
		return nil, err
	}
	if cupom.FiscalDocumentID == nil {
		return nil, &dom.ValidationError{Msg: "o lançamento de origem não é um cupom fiscal"}
	}
	target, err := s.entries.GetByID(ctx, workspaceID, targetEntryID)
	if err != nil {
		return nil, err
	}
	if target.Kind != dom.KindDebit {
		return nil, &dom.ValidationError{Msg: "a compra de destino precisa ser uma despesa"}
	}

	now := time.Now().UTC()

	// 1) Move o detalhamento (itens) do cupom para a compra da fatura.
	if err := s.items.ReassignEntry(ctx, workspaceID, cupomEntryID, targetEntryID); err != nil {
		return nil, err
	}

	// 2) Religa o documento à compra da fatura.
	doc, derr := s.docs.GetByID(ctx, workspaceID, *cupom.FiscalDocumentID)
	if derr == nil {
		doc.EntryID = &targetEntryID
		doc.UpdatedAt = now
		_ = s.docs.UpdateExtraction(ctx, doc)
		target.FiscalDocumentID = &doc.ID
	}
	target.UpdatedAt = now
	if err := s.entries.Update(ctx, target); err != nil {
		return nil, err
	}

	// 3) Remove a despesa avulsa do cupom (desvincula o doc antes p/ segurança).
	cupom.FiscalDocumentID = nil
	cupom.UpdatedAt = now
	_ = s.entries.Update(ctx, cupom)
	if err := s.entries.SoftDelete(ctx, workspaceID, cupomEntryID); err != nil {
		return nil, err
	}

	return target, nil
}

func daysBetween(a, b time.Time) int {
	return int(a.Truncate(24*time.Hour).Sub(b.Truncate(24*time.Hour)).Hours() / 24)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
