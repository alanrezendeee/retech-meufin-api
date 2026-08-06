package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DashboardSummary agrega os números do mês selecionado.
//
// EIXOS TEMPORAIS — um fato, dois eixos, cada medida declara o seu:
//   - Previsto, a pagar e a receber: eixo COMPETÊNCIA (due_date) — a
//     obrigação pertence ao mês em que vence, paga ou não.
//   - Realizado: eixo CAIXA (paid_at, fallback due_date quando a liquidação
//     antiga não gravou a data) — o dinheiro saiu quando saiu. Parcela de
//     dezembro antecipada em agosto conta no realizado de agosto; a atrasada
//     de junho paga em julho conta no realizado de julho.
//
// Regras (fatura pai/filho NÃO pode duplicar):
//   - Totais consideram só lançamentos de topo (parent_id IS NULL).
//   - Categorias consideram só folhas (exclui pai que tem filhos), senão
//     tudo viraria a categoria "cartao"; pode divergir do total quando o
//     amount da fatura foi sobrescrito e difere da soma dos itens.
//   - Cancelados e soft-deleted ficam sempre de fora.
//   - Realizado usa paid_amount_cents quando existir (liquidação com
//     juros/multa/desconto), senão amount_cents.
//   - Previsto = prevista + realizada (total esperado do mês, por vencimento).
type DashboardSummary struct {
	// Cards do mês.
	IncomeRealizedCents  int64
	IncomeExpectedCents  int64
	ExpenseRealizedCents int64
	ExpenseExpectedCents int64
	// Pendências do mês (status prevista).
	ReceivableCents int64
	PayableCents    int64
	// Despesa por categoria (folhas), maior→menor.
	Categories []CategoryTotal
	// Parcelas futuras: débito prevista com installment_number, vencendo após o mês.
	FutureInstallments FutureInstallments
}

type CategoryTotal struct {
	Category   string
	TotalCents int64
}

type FutureInstallments struct {
	TotalCents  int64
	Count       int64
	LastDueDate *time.Time
}

// MonthlyPoint é um mês da série anual (realizado × previsto separados para
// o gráfico mostrar passado real e futuro previsto).
type MonthlyPoint struct {
	Month                int
	IncomeRealizedCents  int64
	IncomeExpectedCents  int64
	ExpenseRealizedCents int64
	ExpenseExpectedCents int64
}

// FinanceDashboardRepository agrega lançamentos em SQL (multitenant).
type FinanceDashboardRepository interface {
	Summary(ctx context.Context, workspaceID uuid.UUID, year, month int, familyMemberID *uuid.UUID) (*DashboardSummary, error)
	MonthlySeries(ctx context.Context, workspaceID uuid.UUID, year int, familyMemberID *uuid.UUID) ([]MonthlyPoint, error)
	// CategoryEntries devolve TODOS os lançamentos que compõem a barra de uma
	// categoria no mês (drill-down) — mesma cláusula da agregação do Summary,
	// sem paginação: o recorte mês × categoria é o limite natural.
	CategoryEntries(ctx context.Context, workspaceID uuid.UUID, category string, year, month int, familyMemberID *uuid.UUID) ([]FinancialEntry, error)
}
