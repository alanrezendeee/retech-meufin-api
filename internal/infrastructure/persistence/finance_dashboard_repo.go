package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

type FinanceDashboardRepository struct {
	db *gorm.DB
}

func NewFinanceDashboardRepository(db *gorm.DB) *FinanceDashboardRepository {
	return &FinanceDashboardRepository{db: db}
}

// Summary agrega os números do mês num único scan (CASE WHEN) + duas queries
// auxiliares (categorias e parcelas futuras). Regras em dom.DashboardSummary.
func (r *FinanceDashboardRepository) Summary(ctx context.Context, workspaceID uuid.UUID, year, month int, familyMemberID *uuid.UUID) (*dom.DashboardSummary, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	type totalsRow struct {
		IncomeRealized  int64
		IncomeExpected  int64
		ExpenseRealized int64
		ExpenseExpected int64
		Receivable      int64
		Payable         int64
	}
	var totals totalsRow

	q := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			COALESCE(SUM(CASE WHEN kind = 'credit' AND status = 'realizada' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS income_realized,
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN amount_cents ELSE 0 END), 0) AS income_expected,
			COALESCE(SUM(CASE WHEN kind = 'debit' AND status = 'realizada' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS expense_realized,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN amount_cents ELSE 0 END), 0) AS expense_expected,
			COALESCE(SUM(CASE WHEN kind = 'credit' AND status = 'prevista' THEN amount_cents ELSE 0 END), 0) AS receivable,
			COALESCE(SUM(CASE WHEN kind = 'debit' AND status = 'prevista' THEN amount_cents ELSE 0 END), 0) AS payable`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status <> 'cancelada'", workspaceID).
		Where("due_date >= ? AND due_date < ?", start, end)
	if familyMemberID != nil {
		q = q.Where("family_member_id = ?", *familyMemberID)
	}
	if err := q.Scan(&totals).Error; err != nil {
		return nil, mapFinanceErr(err)
	}

	// Categorias: só folhas (pai com filhos fica de fora — os filhos carregam
	// as categorias reais). Realizado usa valor pago quando existir.
	type catRow struct {
		Category string
		Total    int64
	}
	var cats []catRow
	cq := r.db.WithContext(ctx).
		Table("financial_entries e").
		Select(`COALESCE(e.type, 'outros') AS category,
			COALESCE(SUM(CASE WHEN e.status = 'realizada' THEN COALESCE(e.paid_amount_cents, e.amount_cents) ELSE e.amount_cents END), 0) AS total`).
		Where("e.workspace_id = ? AND e.deleted_at IS NULL AND e.kind = 'debit' AND e.status <> 'cancelada'", workspaceID).
		Where("e.due_date >= ? AND e.due_date < ?", start, end).
		Where("NOT EXISTS (SELECT 1 FROM financial_entries c WHERE c.parent_id = e.id AND c.deleted_at IS NULL)").
		Group("COALESCE(e.type, 'outros')").
		Order("total DESC")
	if familyMemberID != nil {
		cq = cq.Where("e.family_member_id = ?", *familyMemberID)
	}
	if err := cq.Scan(&cats).Error; err != nil {
		return nil, mapFinanceErr(err)
	}

	// Parcelas futuras: comprometido após o mês selecionado (parcelas manuais
	// já são materializadas na criação; parcelas de fatura importada não).
	type instRow struct {
		Total   int64
		Count   int64
		LastDue *time.Time
	}
	var inst instRow
	iq := r.db.WithContext(ctx).
		Table("financial_entries").
		Select("COALESCE(SUM(amount_cents), 0) AS total, COUNT(*) AS count, MAX(due_date) AS last_due").
		Where("workspace_id = ? AND deleted_at IS NULL AND kind = 'debit' AND status = 'prevista'", workspaceID).
		Where("installment_number IS NOT NULL AND due_date >= ?", end)
	if familyMemberID != nil {
		iq = iq.Where("family_member_id = ?", *familyMemberID)
	}
	if err := iq.Scan(&inst).Error; err != nil {
		return nil, mapFinanceErr(err)
	}

	out := &dom.DashboardSummary{
		IncomeRealizedCents:  totals.IncomeRealized,
		IncomeExpectedCents:  totals.IncomeExpected,
		ExpenseRealizedCents: totals.ExpenseRealized,
		ExpenseExpectedCents: totals.ExpenseExpected,
		ReceivableCents:      totals.Receivable,
		PayableCents:         totals.Payable,
		Categories:           make([]dom.CategoryTotal, len(cats)),
		FutureInstallments: dom.FutureInstallments{
			TotalCents:  inst.Total,
			Count:       inst.Count,
			LastDueDate: inst.LastDue,
		},
	}
	for i := range cats {
		out.Categories[i] = dom.CategoryTotal{Category: cats[i].Category, TotalCents: cats[i].Total}
	}
	return out, nil
}

// MonthlySeries agrega o ano inteiro num scan só, agrupado por mês.
func (r *FinanceDashboardRepository) MonthlySeries(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) ([]dom.MonthlyPoint, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	type row struct {
		Month           int
		IncomeRealized  int64
		IncomeExpected  int64
		ExpenseRealized int64
		ExpenseExpected int64
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			EXTRACT(MONTH FROM due_date)::int AS month,
			COALESCE(SUM(CASE WHEN kind = 'credit' AND status = 'realizada' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS income_realized,
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN amount_cents ELSE 0 END), 0) AS income_expected,
			COALESCE(SUM(CASE WHEN kind = 'debit' AND status = 'realizada' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS expense_realized,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN amount_cents ELSE 0 END), 0) AS expense_expected`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status <> 'cancelada'", workspaceID).
		Where("due_date >= ? AND due_date < ?", start, end).
		Group("EXTRACT(MONTH FROM due_date)").
		Order("month ASC")
	if familyMemberID != nil {
		q = q.Where("family_member_id = ?", *familyMemberID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}

	out := make([]dom.MonthlyPoint, len(rows))
	for i := range rows {
		out[i] = dom.MonthlyPoint{
			Month:                rows[i].Month,
			IncomeRealizedCents:  rows[i].IncomeRealized,
			IncomeExpectedCents:  rows[i].IncomeExpected,
			ExpenseRealizedCents: rows[i].ExpenseRealized,
			ExpenseExpectedCents: rows[i].ExpenseExpected,
		}
	}
	return out, nil
}
