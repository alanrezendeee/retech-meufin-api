package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceFiscalDashboardHandler expõe a análise de cupons/notas fiscais:
// histórico de preço por produto e inflação pessoal. Valores em centavos;
// quantidades em milésimos; o front formata.
type FinanceFiscalDashboardHandler struct {
	svc *app.FiscalDashboardService
}

func NewFinanceFiscalDashboardHandler(svc *app.FiscalDashboardService) *FinanceFiscalDashboardHandler {
	return &FinanceFiscalDashboardHandler{svc: svc}
}

type fiscalProductJSON struct {
	Name           string   `json:"name"`
	Unit           string   `json:"unit"`
	Purchases      int64    `json:"purchases"`
	QtyMilliTotal  int64    `json:"qty_milli_total"`
	TotalCents     int64    `json:"total_cents"`
	AvgUnitCents   int64    `json:"avg_unit_cents"`
	MinUnitCents   int64    `json:"min_unit_cents"`
	MaxUnitCents   int64    `json:"max_unit_cents"`
	FirstUnitCents int64    `json:"first_unit_cents"`
	LastUnitCents  int64    `json:"last_unit_cents"`
	FirstDate      *string  `json:"first_date"`
	LastDate       *string  `json:"last_date"`
	VariationPct   float64  `json:"variation_pct"`
}

func mapFiscalProduct(p app.FiscalProduct) fiscalProductJSON {
	out := fiscalProductJSON{
		Name:           p.Name,
		Unit:           p.Unit,
		Purchases:      p.Purchases,
		QtyMilliTotal:  p.QtyMilliTotal,
		TotalCents:     p.TotalCents,
		AvgUnitCents:   p.AvgUnitCents,
		MinUnitCents:   p.MinUnitCents,
		MaxUnitCents:   p.MaxUnitCents,
		FirstUnitCents: p.FirstUnitCents,
		LastUnitCents:  p.LastUnitCents,
		VariationPct:   p.VariationPct,
	}
	if p.FirstDate != nil {
		v := p.FirstDate.Format(entryDateLayout)
		out.FirstDate = &v
	}
	if p.LastDate != nil {
		v := p.LastDate.Format(entryDateLayout)
		out.LastDate = &v
	}
	return out
}

type fiscalMonthSpendJSON struct {
	Month      string `json:"month"`
	TotalCents int64  `json:"total_cents"`
	Items      int64  `json:"items"`
}

// Summary responde GET /finance/fiscal/dashboard.
func (h *FinanceFiscalDashboardHandler) Summary(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	sum, err := h.svc.Summary(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	byFreq := make([]fiscalProductJSON, len(sum.TopByFrequency))
	for i := range sum.TopByFrequency {
		byFreq[i] = mapFiscalProduct(sum.TopByFrequency[i])
	}
	bySpend := make([]fiscalProductJSON, len(sum.TopBySpend))
	for i := range sum.TopBySpend {
		bySpend[i] = mapFiscalProduct(sum.TopBySpend[i])
	}
	months := make([]fiscalMonthSpendJSON, len(sum.MonthlySpend))
	for i := range sum.MonthlySpend {
		months[i] = fiscalMonthSpendJSON{
			Month:      sum.MonthlySpend[i].Month,
			TotalCents: sum.MonthlySpend[i].TotalCents,
			Items:      sum.MonthlySpend[i].Items,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"documents":       sum.Counts.Documents,
		"items":           sum.Counts.Items,
		"total_cents":     sum.Counts.TotalCents,
		"products_count":  sum.Counts.ProductsCount,
		"top_by_frequency": byFreq,
		"top_by_spend":     bySpend,
		"monthly_spend":    months,
	})
}

// Products responde GET /finance/fiscal/products?q=&sort=&limit=.
func (h *FinanceFiscalDashboardHandler) Products(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	limit := 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "limit inválido")
			return
		}
		limit = n
	}
	products, err := h.svc.Products(c.Request.Context(), ws, c.Query("q"), c.Query("sort"), limit)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]fiscalProductJSON, len(products))
	for i := range products {
		items[i] = mapFiscalProduct(products[i])
	}
	c.JSON(http.StatusOK, gin.H{"products": items, "total": len(items)})
}

type fiscalPurchaseJSON struct {
	PurchaseDate  string `json:"purchase_date"`
	UnitCents     int64  `json:"unit_cents"`
	QuantityMilli int64  `json:"quantity_milli"`
	AmountCents   int64  `json:"amount_cents"`
	Unit          string `json:"unit"`
	DocumentID    string `json:"document_id"`
	DocumentName  string `json:"document_name"`
}

// PriceHistory responde GET /finance/fiscal/products/price-history?name=&unit=.
func (h *FinanceFiscalDashboardHandler) PriceHistory(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	purchases, err := h.svc.PriceHistory(c.Request.Context(), ws, c.Query("name"), c.Query("unit"))
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	items := make([]fiscalPurchaseJSON, len(purchases))
	for i := range purchases {
		items[i] = fiscalPurchaseJSON{
			PurchaseDate:  purchases[i].PurchaseDate.Format(entryDateLayout),
			UnitCents:     purchases[i].UnitCents,
			QuantityMilli: purchases[i].QuantityMilli,
			AmountCents:   purchases[i].AmountCents,
			Unit:          purchases[i].Unit,
			DocumentID:    purchases[i].DocumentID.String(),
			DocumentName:  purchases[i].DocumentName,
		}
	}
	c.JSON(http.StatusOK, gin.H{"name": c.Query("name"), "unit": c.Query("unit"), "purchases": items, "total": len(items)})
}

type fiscalInflationPointJSON struct {
	Month           string  `json:"month"`
	Index           float64 `json:"index"`
	MonthlyPct      float64 `json:"monthly_pct"`
	MatchedProducts int     `json:"matched_products"`
}

// Inflation responde GET /finance/fiscal/inflation.
func (h *FinanceFiscalDashboardHandler) Inflation(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	inf, err := h.svc.Inflation(c.Request.Context(), ws)
	if err != nil {
		errrespond.Write(c, err)
		return
	}
	points := make([]fiscalInflationPointJSON, len(inf.Points))
	for i := range inf.Points {
		points[i] = fiscalInflationPointJSON{
			Month:           inf.Points[i].Month,
			Index:           inf.Points[i].Index,
			MonthlyPct:      inf.Points[i].MonthlyPct,
			MatchedProducts: inf.Points[i].MatchedProducts,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"points":        points,
		"variation_12m": inf.Variation12m,
		"methodology":   inf.Methodology,
	})
}
