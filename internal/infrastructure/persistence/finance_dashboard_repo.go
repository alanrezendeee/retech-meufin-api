package persistence

import (
	"context"
	"sort"
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

	// Eixo COMPETÊNCIA (due_date): previsto e pendências — a obrigação
	// pertence ao mês em que vence, paga ou não.
	type expectedRow struct {
		IncomeExpected  int64
		ExpenseExpected int64
		Receivable      int64
		Payable         int64
	}
	var expected expectedRow
	q := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN amount_cents ELSE 0 END), 0) AS income_expected,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN amount_cents ELSE 0 END), 0) AS expense_expected,
			COALESCE(SUM(CASE WHEN kind = 'credit' AND status = 'prevista' THEN amount_cents ELSE 0 END), 0) AS receivable,
			COALESCE(SUM(CASE WHEN kind = 'debit' AND status = 'prevista' THEN amount_cents ELSE 0 END), 0) AS payable`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status <> 'cancelada'", workspaceID).
		Where("due_date >= ? AND due_date < ?", start, end)
	if familyMemberID != nil {
		q = q.Where("family_member_id = ?", *familyMemberID)
	}
	if err := q.Scan(&expected).Error; err != nil {
		return nil, mapFinanceErr(err)
	}

	// Eixo CAIXA (paid_at, fallback due_date para liquidações antigas sem a
	// data): realizado — o dinheiro saiu quando saiu. É o que faz a parcela
	// antecipada aparecer no mês do pagamento, não no do boleto.
	type realizedRow struct {
		IncomeRealized  int64
		ExpenseRealized int64
	}
	var realized realizedRow
	rq := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS income_realized,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS expense_realized`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status = 'realizada'", workspaceID).
		Where("COALESCE(DATE(paid_at), due_date) >= ? AND COALESCE(DATE(paid_at), due_date) < ?", start, end)
	if familyMemberID != nil {
		rq = rq.Where("family_member_id = ?", *familyMemberID)
	}
	if err := rq.Scan(&realized).Error; err != nil {
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
		// Cada status no seu eixo: prevista por vencimento (competência),
		// realizada por pagamento (caixa, fallback vencimento).
		Where(`((e.status = 'prevista' AND e.due_date >= ? AND e.due_date < ?)
		    OR (e.status = 'realizada' AND COALESCE(DATE(e.paid_at), e.due_date) >= ? AND COALESCE(DATE(e.paid_at), e.due_date) < ?))`,
			start, end, start, end).
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
		IncomeRealizedCents:  realized.IncomeRealized,
		IncomeExpectedCents:  expected.IncomeExpected,
		ExpenseRealizedCents: realized.ExpenseRealized,
		ExpenseExpectedCents: expected.ExpenseExpected,
		ReceivableCents:      expected.Receivable,
		PayableCents:         expected.Payable,
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
// CategoryEntries materializa a barra do "Pra onde foi o dinheiro": os
// lançamentos por trás da agregação de categorias do Summary. A cláusula é
// deliberadamente IDÊNTICA à do agregado — só folhas (fatura pai com itens
// fica de fora; os itens contam pela própria categoria), canceladas fora,
// recorte por due_date no mês. Se as duas divergirem, o modal não bate com a
// barra e o usuário perde a confiança no número.
func (r *FinanceDashboardRepository) CategoryEntries(ctx context.Context, workspaceID uuid.UUID, category string, year, month int, familyMemberID *uuid.UUID) ([]dom.FinancialEntry, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	q := r.db.WithContext(ctx).
		Table("financial_entries e").
		Select("e.*").
		Where("e.workspace_id = ? AND e.deleted_at IS NULL AND e.kind = 'debit' AND e.status <> 'cancelada'", workspaceID).
		// Espelho exato da agregação: prevista por vencimento, realizada por
		// pagamento — senão o modal não bate com a barra.
		Where(`((e.status = 'prevista' AND e.due_date >= ? AND e.due_date < ?)
		    OR (e.status = 'realizada' AND COALESCE(DATE(e.paid_at), e.due_date) >= ? AND COALESCE(DATE(e.paid_at), e.due_date) < ?))`,
			start, end, start, end).
		Where("NOT EXISTS (SELECT 1 FROM financial_entries c WHERE c.parent_id = e.id AND c.deleted_at IS NULL)").
		Where("COALESCE(e.type, 'outros') = ?", category).
		Order("e.due_date ASC, e.created_at ASC")
	if familyMemberID != nil {
		q = q.Where("e.family_member_id = ?", *familyMemberID)
	}

	var rows []FinancialEntryModel
	if err := q.Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
}

func (r *FinanceDashboardRepository) MonthlySeries(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) ([]dom.MonthlyPoint, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	// Um lançamento pode contribuir para MESES DIFERENTES nas duas medidas:
	// a parcela de dezembro antecipada em agosto soma no previsto de dezembro
	// (competência) e no realizado de agosto (caixa). Por isso são dois
	// GROUP BY — um por eixo — mesclados em memória.
	byMonth := map[int]*dom.MonthlyPoint{}
	point := func(m int) *dom.MonthlyPoint {
		if p, ok := byMonth[m]; ok {
			return p
		}
		p := &dom.MonthlyPoint{Month: m}
		byMonth[m] = p
		return p
	}

	type expRow struct {
		Month           int
		IncomeExpected  int64
		ExpenseExpected int64
	}
	var expRows []expRow
	q := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			EXTRACT(MONTH FROM due_date)::int AS month,
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN amount_cents ELSE 0 END), 0) AS income_expected,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN amount_cents ELSE 0 END), 0) AS expense_expected`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status <> 'cancelada'", workspaceID).
		Where("due_date >= ? AND due_date < ?", start, end).
		Group("EXTRACT(MONTH FROM due_date)")
	if familyMemberID != nil {
		q = q.Where("family_member_id = ?", *familyMemberID)
	}
	if err := q.Scan(&expRows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	for i := range expRows {
		p := point(expRows[i].Month)
		p.IncomeExpectedCents = expRows[i].IncomeExpected
		p.ExpenseExpectedCents = expRows[i].ExpenseExpected
	}

	type realRow struct {
		Month           int
		IncomeRealized  int64
		ExpenseRealized int64
	}
	var realRows []realRow
	rq := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			EXTRACT(MONTH FROM COALESCE(DATE(paid_at), due_date))::int AS month,
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS income_realized,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS expense_realized`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status = 'realizada'", workspaceID).
		Where("COALESCE(DATE(paid_at), due_date) >= ? AND COALESCE(DATE(paid_at), due_date) < ?", start, end).
		Group("EXTRACT(MONTH FROM COALESCE(DATE(paid_at), due_date))")
	if familyMemberID != nil {
		rq = rq.Where("family_member_id = ?", *familyMemberID)
	}
	if err := rq.Scan(&realRows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	for i := range realRows {
		p := point(realRows[i].Month)
		p.IncomeRealizedCents = realRows[i].IncomeRealized
		p.ExpenseRealizedCents = realRows[i].ExpenseRealized
	}

	months := make([]int, 0, len(byMonth))
	for m := range byMonth {
		months = append(months, m)
	}
	sort.Ints(months)
	out := make([]dom.MonthlyPoint, 0, len(months))
	for _, m := range months {
		out = append(out, *byMonth[m])
	}
	return out, nil
}

// CashFlowRaw agrega os fluxos realizados no eixo caixa.
//
// Abertura = líquido (créditos − débitos) de tudo que foi pago antes de 1º de
// janeiro do ano — o caixa que o histórico registrado construiu até ali.
// Mesmas exclusões do resto do dashboard: topo apenas (fatura conta pelo pai)
// e soft-deleted fora. Realizada é o único status que move caixa.
func (r *FinanceDashboardRepository) CashFlowRaw(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) (int64, []dom.CashFlowMonth, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	var opening struct{ Net int64 }
	oq := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`COALESCE(SUM(CASE WHEN kind = 'credit' THEN COALESCE(paid_amount_cents, amount_cents)
		                          ELSE -COALESCE(paid_amount_cents, amount_cents) END), 0) AS net`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status = 'realizada'", workspaceID).
		Where("COALESCE(DATE(paid_at), due_date) < ?", start)
	if familyMemberID != nil {
		oq = oq.Where("family_member_id = ?", *familyMemberID)
	}
	if err := oq.Scan(&opening).Error; err != nil {
		return 0, nil, mapFinanceErr(err)
	}

	type row struct {
		Month   int
		Inflow  int64
		Outflow int64
	}
	var rows []row
	mq := r.db.WithContext(ctx).
		Table("financial_entries").
		Select(`
			EXTRACT(MONTH FROM COALESCE(DATE(paid_at), due_date))::int AS month,
			COALESCE(SUM(CASE WHEN kind = 'credit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS inflow,
			COALESCE(SUM(CASE WHEN kind = 'debit' THEN COALESCE(paid_amount_cents, amount_cents) ELSE 0 END), 0) AS outflow`).
		Where("workspace_id = ? AND deleted_at IS NULL AND parent_id IS NULL AND status = 'realizada'", workspaceID).
		Where("COALESCE(DATE(paid_at), due_date) >= ? AND COALESCE(DATE(paid_at), due_date) < ?", start, end).
		Group("EXTRACT(MONTH FROM COALESCE(DATE(paid_at), due_date))").
		Order("month ASC")
	if familyMemberID != nil {
		mq = mq.Where("family_member_id = ?", *familyMemberID)
	}
	if err := mq.Scan(&rows).Error; err != nil {
		return 0, nil, mapFinanceErr(err)
	}

	out := make([]dom.CashFlowMonth, len(rows))
	for i := range rows {
		out[i] = dom.CashFlowMonth{
			Month:        rows[i].Month,
			InflowCents:  rows[i].Inflow,
			OutflowCents: rows[i].Outflow,
		}
	}
	return opening.Net, out, nil
}
