package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	appf "github.com/retechfin/retechfin-api/internal/application/finance"
	"gorm.io/gorm"
)

// FiscalDashboardRepository agrega finance_fiscal_items para o painel fiscal.
// Os itens são detalhamento informacional; a data da compra vem do lançamento
// vinculado (purchase_date quando informada, senão due_date da fatura/despesa).
type FiscalDashboardRepository struct{ db *gorm.DB }

func NewFiscalDashboardRepository(db *gorm.DB) *FiscalDashboardRepository {
	return &FiscalDashboardRepository{db: db}
}

// unitPriceExpr é o preço unitário efetivo em centavos: usa unit_cents quando
// preenchido; senão deriva de amount_cents ÷ quantidade (quantity_milli/1000);
// no limite cai no próprio amount_cents.
const unitPriceExpr = `CASE
	WHEN fi.unit_cents > 0 THEN fi.unit_cents
	WHEN fi.quantity_milli > 0 THEN ROUND(fi.amount_cents * 1000.0 / fi.quantity_milli)
	ELSE fi.amount_cents END`

// purchaseDateExpr é a data da compra: purchase_date do lançamento quando
// informada, senão o vencimento (due_date).
const purchaseDateExpr = `COALESCE(e.purchase_date, e.due_date)`

// Identidade do produto = (nome normalizado, unidade de medida normalizada).
// Separar por unidade impede misturar R$/kg com R$/un no mesmo "produto".
const productNameExpr = `LOWER(TRIM(fi.description))`
const productUnitExpr = `UPPER(TRIM(COALESCE(fi.unit_of_measure, '')))`

// fiscalJoin é o join base itens→lançamento, escopado por workspace e ignorando
// lançamentos deletados. Sempre parametrize workspace com fi.workspace_id = ?.
const fiscalJoin = `FROM finance_fiscal_items fi
	JOIN financial_entries e ON e.id = fi.entry_id AND e.workspace_id = fi.workspace_id AND e.deleted_at IS NULL`

// Counts retorna os números-resumo do painel.
func (r *FiscalDashboardRepository) Counts(ctx context.Context, workspaceID uuid.UUID) (*appf.FiscalDashboardCounts, error) {
	var row struct {
		Documents     int64
		Items         int64
		TotalCents    int64
		ProductsCount int64
	}
	sql := `SELECT
		COUNT(DISTINCT fi.document_id) AS documents,
		COUNT(*) AS items,
		COALESCE(SUM(fi.amount_cents), 0) AS total_cents,
		COUNT(DISTINCT (` + productNameExpr + `, ` + productUnitExpr + `)) AS products_count
	` + fiscalJoin + `
	WHERE fi.workspace_id = ?`
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID).Scan(&row).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	return &appf.FiscalDashboardCounts{
		Documents:     row.Documents,
		Items:         row.Items,
		TotalCents:    row.TotalCents,
		ProductsCount: row.ProductsCount,
	}, nil
}

// MonthlySpend soma o gasto fiscal por mês a partir de since (inclusive).
func (r *FiscalDashboardRepository) MonthlySpend(ctx context.Context, workspaceID uuid.UUID, since time.Time) ([]appf.FiscalMonthSpend, error) {
	var rows []struct {
		Month      string
		TotalCents int64
		Items      int64
	}
	sql := `SELECT
		to_char(date_trunc('month', ` + purchaseDateExpr + `), 'YYYY-MM') AS month,
		COALESCE(SUM(fi.amount_cents), 0) AS total_cents,
		COUNT(*) AS items
	` + fiscalJoin + `
	WHERE fi.workspace_id = ? AND ` + purchaseDateExpr + ` >= ?
	GROUP BY 1
	ORDER BY 1`
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID, since).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.FiscalMonthSpend, len(rows))
	for i := range rows {
		out[i] = appf.FiscalMonthSpend{Month: rows[i].Month, TotalCents: rows[i].TotalCents, Items: rows[i].Items}
	}
	return out, nil
}

// TopProducts agrega produtos por nome normalizado. sortBy define a ordenação
// (whitelist interna, nunca interpolada do usuário direto); q é substring já
// normalizada (vazia = sem filtro); limit ≤ 0 = sem limite.
func (r *FiscalDashboardRepository) TopProducts(ctx context.Context, workspaceID uuid.UUID, q, sortBy string, limit int) ([]appf.FiscalProduct, error) {
	orderCol := "purchases DESC, total_cents DESC"
	switch sortBy {
	case "spend":
		orderCol = "total_cents DESC, purchases DESC"
	case "inflation":
		orderCol = "variation_pct DESC, total_cents DESC"
	case "frequency":
		orderCol = "purchases DESC, total_cents DESC"
	}
	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", limit)
	}

	sql := `WITH agg AS (
		SELECT
			` + productNameExpr + ` AS name,
			` + productUnitExpr + ` AS unit,
			COUNT(*) AS purchases,
			COALESCE(SUM(fi.quantity_milli), 0) AS qty_milli_total,
			COALESCE(SUM(fi.amount_cents), 0) AS total_cents,
			ROUND(AVG(` + unitPriceExpr + `))::bigint AS avg_unit_cents,
			MIN(` + unitPriceExpr + `)::bigint AS min_unit_cents,
			MAX(` + unitPriceExpr + `)::bigint AS max_unit_cents,
			MIN(` + purchaseDateExpr + `) AS first_date,
			MAX(` + purchaseDateExpr + `) AS last_date,
			(array_agg((` + unitPriceExpr + `)::bigint ORDER BY ` + purchaseDateExpr + ` ASC, fi.created_at ASC))[1] AS first_unit_cents,
			(array_agg((` + unitPriceExpr + `)::bigint ORDER BY ` + purchaseDateExpr + ` DESC, fi.created_at DESC))[1] AS last_unit_cents
		` + fiscalJoin + `
		WHERE fi.workspace_id = ?
		GROUP BY ` + productNameExpr + `, ` + productUnitExpr + `
	)
	SELECT *,
		CASE WHEN first_unit_cents > 0
			THEN ROUND(((last_unit_cents - first_unit_cents)::numeric / first_unit_cents) * 100, 2)
			ELSE 0 END AS variation_pct
	FROM agg
	WHERE (? = '' OR name LIKE ?)
	ORDER BY ` + orderCol + limitClause

	like := "%" + q + "%"
	var rows []struct {
		Name           string
		Unit           string
		Purchases      int64
		QtyMilliTotal  int64
		TotalCents     int64
		AvgUnitCents   int64
		MinUnitCents   int64
		MaxUnitCents   int64
		FirstDate      *time.Time
		LastDate       *time.Time
		FirstUnitCents int64
		LastUnitCents  int64
		VariationPct   float64
	}
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID, q, like).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.FiscalProduct, len(rows))
	for i := range rows {
		out[i] = appf.FiscalProduct{
			Name:           rows[i].Name,
			Unit:           rows[i].Unit,
			Purchases:      rows[i].Purchases,
			QtyMilliTotal:  rows[i].QtyMilliTotal,
			TotalCents:     rows[i].TotalCents,
			AvgUnitCents:   rows[i].AvgUnitCents,
			MinUnitCents:   rows[i].MinUnitCents,
			MaxUnitCents:   rows[i].MaxUnitCents,
			FirstUnitCents: rows[i].FirstUnitCents,
			LastUnitCents:  rows[i].LastUnitCents,
			FirstDate:      rows[i].FirstDate,
			LastDate:       rows[i].LastDate,
			VariationPct:   rows[i].VariationPct,
		}
	}
	return out, nil
}

// PriceHistory retorna a série cronológica de compras de um produto (nome já
// normalizado), com o nome do documento de origem quando disponível.
func (r *FiscalDashboardRepository) PriceHistory(ctx context.Context, workspaceID uuid.UUID, name, unit string) ([]appf.FiscalPurchase, error) {
	sql := `SELECT
		` + purchaseDateExpr + ` AS purchase_date,
		(` + unitPriceExpr + `)::bigint AS unit_cents,
		fi.quantity_milli AS quantity_milli,
		fi.amount_cents AS amount_cents,
		` + productUnitExpr + ` AS unit,
		fi.document_id AS document_id,
		COALESCE(d.original_file_name, '') AS document_name
	` + fiscalJoin + `
	LEFT JOIN finance_documents d ON d.id = fi.document_id AND d.workspace_id = fi.workspace_id AND d.deleted_at IS NULL
	WHERE fi.workspace_id = ? AND ` + productNameExpr + ` = ? AND ` + productUnitExpr + ` = ?
	ORDER BY purchase_date ASC, fi.created_at ASC`
	var rows []struct {
		PurchaseDate  time.Time
		UnitCents     int64
		QuantityMilli int64
		AmountCents   int64
		Unit          string
		DocumentID    uuid.UUID
		DocumentName  string
	}
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID, name, unit).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.FiscalPurchase, len(rows))
	for i := range rows {
		out[i] = appf.FiscalPurchase{
			PurchaseDate:  rows[i].PurchaseDate,
			UnitCents:     rows[i].UnitCents,
			QuantityMilli: rows[i].QuantityMilli,
			AmountCents:   rows[i].AmountCents,
			Unit:          rows[i].Unit,
			DocumentID:    rows[i].DocumentID,
			DocumentName:  rows[i].DocumentName,
		}
	}
	return out, nil
}

// MonthlyProductPrices retorna o preço unitário médio (total gasto ÷ quantidade)
// de cada produto por mês — insumo do índice de inflação pessoal.
func (r *FiscalDashboardRepository) MonthlyProductPrices(ctx context.Context, workspaceID uuid.UUID) ([]appf.MonthlyProductPrice, error) {
	sql := `SELECT
		to_char(date_trunc('month', ` + purchaseDateExpr + `), 'YYYY-MM') AS month,
		` + productNameExpr + ` AS name,
		` + productUnitExpr + ` AS unit,
		CASE WHEN SUM(fi.quantity_milli) > 0
			THEN ROUND(SUM(fi.amount_cents) * 1000.0 / SUM(fi.quantity_milli))::bigint
			ELSE ROUND(AVG(` + unitPriceExpr + `))::bigint END AS avg_unit_cents,
		COALESCE(SUM(fi.amount_cents), 0) AS spend_cents,
		COUNT(*) AS purchases
	` + fiscalJoin + `
	WHERE fi.workspace_id = ?
	GROUP BY 1, 2, 3
	ORDER BY 1 ASC`
	var rows []struct {
		Month        string
		Name         string
		Unit         string
		AvgUnitCents int64
		SpendCents   int64
		Purchases    int64
	}
	if err := r.db.WithContext(ctx).Raw(sql, workspaceID).Scan(&rows).Error; err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]appf.MonthlyProductPrice, len(rows))
	for i := range rows {
		out[i] = appf.MonthlyProductPrice{
			Month:        rows[i].Month,
			Name:         rows[i].Name,
			Unit:         rows[i].Unit,
			AvgUnitCents: rows[i].AvgUnitCents,
			SpendCents:   rows[i].SpendCents,
			Purchases:    rows[i].Purchases,
		}
	}
	return out, nil
}
