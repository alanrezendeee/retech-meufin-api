package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	appf "github.com/retechfin/retechfin-api/internal/application/finance"
	"gorm.io/gorm"
)

// ReconciliationRepository consulta compras de fatura e cupons conciliáveis
// (despesas avulsas de cupom pago no crédito). Escopo por workspace.
type ReconciliationRepository struct{ db *gorm.DB }

func NewReconciliationRepository(db *gorm.DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

// InvoicePurchases lista as compras (débitos filhos) de uma fatura.
func (r *ReconciliationRepository) InvoicePurchases(ctx context.Context, workspaceID, invoiceEntryID uuid.UUID) ([]appf.InvoicePurchase, error) {
	var rows []struct {
		ID          uuid.UUID
		AmountCents int64
		Date        time.Time
		Description string
	}
	sql := `SELECT id, amount_cents, COALESCE(purchase_date, due_date) AS date, description
		FROM financial_entries
		WHERE workspace_id = ? AND deleted_at IS NULL AND kind = 'debit' AND parent_id = ?
		ORDER BY amount_cents DESC`
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID, invoiceEntryID).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.InvoicePurchase, len(rows))
	for i := range rows {
		out[i] = appf.InvoicePurchase{
			EntryID:     rows[i].ID,
			AmountCents: rows[i].AmountCents,
			Date:        rows[i].Date,
			Description: rows[i].Description,
		}
	}
	return out, nil
}

// ReconcilableCupons lista despesas avulsas (fora de fatura) originadas de um
// cupom fiscal pago no CRÉDITO — candidatas a conciliar com a fatura.
func (r *ReconciliationRepository) ReconcilableCupons(ctx context.Context, workspaceID uuid.UUID) ([]appf.ReconcileCandidate, error) {
	var rows []struct {
		CupomEntryID uuid.UUID
		DocumentID   uuid.UUID
		AmountCents  int64
		Date         time.Time
		Merchant     string
	}
	sql := `SELECT e.id AS cupom_entry_id, d.id AS document_id, e.amount_cents,
			COALESCE(e.purchase_date, e.due_date) AS date, e.description AS merchant
		FROM financial_entries e
		JOIN finance_documents d ON d.id = e.fiscal_document_id
			AND d.workspace_id = e.workspace_id AND d.deleted_at IS NULL
			AND d.kind = 'fiscal' AND d.payment_method = 'credito'
		WHERE e.workspace_id = ? AND e.deleted_at IS NULL AND e.kind = 'debit'
			AND e.parent_id IS NULL`
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.ReconcileCandidate, len(rows))
	for i := range rows {
		out[i] = appf.ReconcileCandidate{
			CupomEntryID: rows[i].CupomEntryID,
			DocumentID:   rows[i].DocumentID,
			AmountCents:  rows[i].AmountCents,
			Date:         rows[i].Date,
			Merchant:     rows[i].Merchant,
		}
	}
	return out, nil
}
