package finance

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// normalizeName aplica a mesma normalização do agrupamento SQL
// (LOWER(TRIM(description))) para busca por nome de produto.
func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// FiscalDashboardService responde à análise de cupons/notas fiscais: histórico
// de preço por produto e índice de inflação pessoal. Trabalha sobre os itens
// já persistidos (finance_fiscal_items), que são informação — nada aqui altera
// saldos. Todos os valores em centavos; quantidades em milésimos.
type FiscalDashboardService struct {
	repo FiscalDashboardRepository
}

func NewFiscalDashboardService(repo FiscalDashboardRepository) *FiscalDashboardService {
	return &FiscalDashboardService{repo: repo}
}

// FiscalDashboardCounts são os números-resumo do painel fiscal.
type FiscalDashboardCounts struct {
	Documents     int64 // documentos fiscais distintos com itens
	Items         int64 // total de itens
	TotalCents    int64 // total gasto (soma dos itens)
	ProductsCount int64 // produtos distintos (nome normalizado)
}

// FiscalProduct é um produto agregado por nome normalizado (LOWER(TRIM(desc))).
type FiscalProduct struct {
	Name          string
	Purchases     int64 // nº de compras (linhas de item)
	QtyMilliTotal int64 // quantidade total em milésimos
	TotalCents    int64 // gasto total no produto
	AvgUnitCents  int64 // preço unitário médio
	MinUnitCents  int64
	MaxUnitCents  int64
	FirstUnitCents int64      // preço unitário na primeira compra
	LastUnitCents  int64      // preço unitário na última compra
	FirstDate      *time.Time // data da primeira compra
	LastDate       *time.Time // data da última compra
	VariationPct   float64    // variação % primeiro→último preço unitário
}

// FiscalMonthSpend é o gasto fiscal de um mês (bucket YYYY-MM).
type FiscalMonthSpend struct {
	Month      string // YYYY-MM
	TotalCents int64
	Items      int64
}

// FiscalPurchase é uma compra individual de um produto (uma linha de item).
type FiscalPurchase struct {
	PurchaseDate  time.Time
	UnitCents     int64
	QuantityMilli int64
	AmountCents   int64
	DocumentID    uuid.UUID
	DocumentName  string // nome do arquivo de origem (proxy do emissor/loja)
}

// MonthlyProductPrice é o preço unitário médio de um produto num mês — insumo
// bruto do índice de inflação pessoal.
type MonthlyProductPrice struct {
	Month        string // YYYY-MM
	Name         string
	AvgUnitCents int64
	SpendCents   int64
	Purchases    int64
}

// FiscalDashboardRepository agrega itens fiscais por produto e por mês. Todas as
// queries são escopadas por workspace e ignoram lançamentos deletados.
type FiscalDashboardRepository interface {
	Counts(ctx context.Context, workspaceID uuid.UUID) (*FiscalDashboardCounts, error)
	MonthlySpend(ctx context.Context, workspaceID uuid.UUID, since time.Time) ([]FiscalMonthSpend, error)
	// TopProducts lista produtos agregados. q filtra por nome (substring, já
	// normalizada); sort ∈ {frequency,spend,inflation}; limit ≤ 0 = sem limite.
	TopProducts(ctx context.Context, workspaceID uuid.UUID, q, sortBy string, limit int) ([]FiscalProduct, error)
	PriceHistory(ctx context.Context, workspaceID uuid.UUID, name string) ([]FiscalPurchase, error)
	MonthlyProductPrices(ctx context.Context, workspaceID uuid.UUID) ([]MonthlyProductPrice, error)
}

// FiscalDashboardSummary é a resposta do painel-resumo.
type FiscalDashboardSummary struct {
	Counts        FiscalDashboardCounts
	TopByFrequency []FiscalProduct
	TopBySpend     []FiscalProduct
	MonthlySpend   []FiscalMonthSpend // últimos 13 meses, sempre preenchido
}

// Summary monta o resumo: contagens, top 10 por frequência, top 10 por gasto e
// o gasto fiscal dos últimos 13 meses (buckets vazios preenchidos com zero).
func (s *FiscalDashboardService) Summary(ctx context.Context, workspaceID uuid.UUID) (*FiscalDashboardSummary, error) {
	counts, err := s.repo.Counts(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	byFreq, err := s.repo.TopProducts(ctx, workspaceID, "", "frequency", 10)
	if err != nil {
		return nil, err
	}
	bySpend, err := s.repo.TopProducts(ctx, workspaceID, "", "spend", 10)
	if err != nil {
		return nil, err
	}
	// Janela de 13 meses fechada no primeiro dia do mês corrente.
	now := time.Now().UTC()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	since := firstOfMonth.AddDate(0, -12, 0)
	spend, err := s.repo.MonthlySpend(ctx, workspaceID, since)
	if err != nil {
		return nil, err
	}
	return &FiscalDashboardSummary{
		Counts:         *counts,
		TopByFrequency: byFreq,
		TopBySpend:     bySpend,
		MonthlySpend:   fillMonthlySpend(spend, since, 13),
	}, nil
}

// Products lista produtos agregados com busca e ordenação.
func (s *FiscalDashboardService) Products(ctx context.Context, workspaceID uuid.UUID, q, sortBy string, limit int) ([]FiscalProduct, error) {
	switch sortBy {
	case "frequency", "spend", "inflation":
	case "":
		sortBy = "frequency"
	default:
		return nil, &dom.ValidationError{Msg: "sort inválido (use frequency, spend ou inflation)"}
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	return s.repo.TopProducts(ctx, workspaceID, normalizeName(q), sortBy, limit)
}

// PriceHistory retorna a série de compras de um produto (nome normalizado).
func (s *FiscalDashboardService) PriceHistory(ctx context.Context, workspaceID uuid.UUID, name string) ([]FiscalPurchase, error) {
	name = normalizeName(name)
	if name == "" {
		return nil, &dom.ValidationError{Msg: "name é obrigatório"}
	}
	return s.repo.PriceHistory(ctx, workspaceID, name)
}

// FiscalInflationPoint é um ponto do índice de inflação pessoal.
type FiscalInflationPoint struct {
	Month           string  // YYYY-MM
	Index           float64 // base 100 no mês mais antigo com dados
	MonthlyPct      float64 // variação % do fator no mês (encadeado)
	MatchedProducts int     // nº de produtos casados com o mês anterior
}

// FiscalInflation é o índice de inflação pessoal e sua metodologia.
type FiscalInflation struct {
	Points        []FiscalInflationPoint
	Variation12m  float64 // variação % do índice nos últimos 12 meses (ou desde o início)
	Methodology   string
}

const inflationMethodology = "Índice de inflação pessoal por produto, encadeado e ponderado pelo gasto (Laspeyres encadeado). " +
	"Para cada mês com itens fiscais, calcula-se o preço unitário médio de cada produto (total gasto ÷ quantidade). " +
	"O fator de inflação do mês é a média das razões preço_mês / preço_mês_anterior dos produtos presentes em ambos os meses (portanto com 2+ compras), " +
	"ponderada pelo gasto do produto no mês corrente. O índice acumulado parte de 100 no mês mais antigo com dados e é multiplicado pelo fator a cada mês. " +
	"Meses sem produtos casados mantêm o índice (fator 1). A variação 12 meses compara o índice do último mês com o de 12 meses antes (ou o mais antigo disponível)."

// Inflation calcula o índice de inflação pessoal mensal. A metodologia está
// documentada em inflationMethodology e devolvida no campo Methodology.
func (s *FiscalDashboardService) Inflation(ctx context.Context, workspaceID uuid.UUID) (*FiscalInflation, error) {
	rows, err := s.repo.MonthlyProductPrices(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return buildInflation(rows), nil
}

// byMonth agrupa preços de produtos por mês preservando a ordem cronológica.
func buildInflation(rows []MonthlyProductPrice) *FiscalInflation {
	out := &FiscalInflation{Methodology: inflationMethodology, Points: []FiscalInflationPoint{}}
	if len(rows) == 0 {
		return out
	}
	// month -> (name -> price/spend)
	type pp struct {
		unit  int64
		spend int64
	}
	perMonth := map[string]map[string]pp{}
	months := []string{}
	for _, row := range rows {
		m, ok := perMonth[row.Month]
		if !ok {
			m = map[string]pp{}
			perMonth[row.Month] = m
			months = append(months, row.Month)
		}
		m[row.Name] = pp{unit: row.AvgUnitCents, spend: row.SpendCents}
	}
	sort.Strings(months)

	idx := 100.0
	for i, month := range months {
		point := FiscalInflationPoint{Month: month, Index: idx, MonthlyPct: 0}
		if i > 0 {
			prev := perMonth[months[i-1]]
			cur := perMonth[month]
			var wSum, factorSum float64
			for name, curPP := range cur {
				prevPP, ok := prev[name]
				if !ok || prevPP.unit <= 0 || curPP.unit <= 0 {
					continue
				}
				ratio := float64(curPP.unit) / float64(prevPP.unit)
				w := float64(curPP.spend)
				if w <= 0 {
					w = 1
				}
				factorSum += ratio * w
				wSum += w
				point.MatchedProducts++
			}
			factor := 1.0
			if wSum > 0 {
				factor = factorSum / wSum
			}
			idx *= factor
			point.Index = idx
			point.MonthlyPct = (factor - 1) * 100
		}
		out.Points = append(out.Points, point)
	}

	// Variação 12 meses: índice do último vs. 12 meses antes (ou o mais antigo).
	last := out.Points[len(out.Points)-1]
	base := out.Points[0]
	if len(out.Points) > 12 {
		base = out.Points[len(out.Points)-13]
	}
	if base.Index > 0 {
		out.Variation12m = (last.Index/base.Index - 1) * 100
	}
	return out
}

// fillMonthlySpend garante count buckets consecutivos a partir de since (mês a
// mês), preenchendo com zero os meses sem gasto fiscal.
func fillMonthlySpend(rows []FiscalMonthSpend, since time.Time, count int) []FiscalMonthSpend {
	byMonth := make(map[string]FiscalMonthSpend, len(rows))
	for _, r := range rows {
		byMonth[r.Month] = r
	}
	out := make([]FiscalMonthSpend, count)
	cursor := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		key := cursor.Format("2006-01")
		if r, ok := byMonth[key]; ok {
			out[i] = r
		} else {
			out[i] = FiscalMonthSpend{Month: key}
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return out
}
