package finance

import (
	"testing"
	"time"
)

func TestBuildInflationChainedWeighted(t *testing.T) {
	// picanha: 5000 -> 6000 (+20%); arroz: 1000 -> 1000 (0%).
	// Mês 2 pondera pelo gasto do mês corrente: picanha gasta 12000, arroz 2000.
	// fator = (1.20*12000 + 1.00*2000) / 14000 = (14400+2000)/14000 ≈ 1.1714
	rows := []MonthlyProductPrice{
		{Month: "2026-01", Name: "picanha", AvgUnitCents: 5000, SpendCents: 10000},
		{Month: "2026-01", Name: "arroz", AvgUnitCents: 1000, SpendCents: 2000},
		{Month: "2026-02", Name: "picanha", AvgUnitCents: 6000, SpendCents: 12000},
		{Month: "2026-02", Name: "arroz", AvgUnitCents: 1000, SpendCents: 2000},
	}
	inf := buildInflation(rows)
	if len(inf.Points) != 2 {
		t.Fatalf("esperava 2 pontos, veio %d", len(inf.Points))
	}
	if inf.Points[0].Index != 100 {
		t.Fatalf("índice base deve ser 100, veio %v", inf.Points[0].Index)
	}
	if inf.Points[1].MatchedProducts != 2 {
		t.Fatalf("esperava 2 produtos casados, veio %d", inf.Points[1].MatchedProducts)
	}
	wantFactor := (1.20*12000 + 1.00*2000) / 14000
	wantIndex := 100 * wantFactor
	if diff := inf.Points[1].Index - wantIndex; diff > 0.01 || diff < -0.01 {
		t.Fatalf("índice mês 2 esperado ≈%.4f, veio %.4f", wantIndex, inf.Points[1].Index)
	}
	if inf.Methodology == "" {
		t.Fatal("methodology não pode ser vazia")
	}
}

func TestBuildInflationSeparaPorUnidade(t *testing.T) {
	// Mesmo NOME "tomate" com unidades diferentes (KG e UN) NÃO deve casar:
	// são produtos distintos. Só o par (tomate, KG) casa entre os meses.
	rows := []MonthlyProductPrice{
		{Month: "2026-01", Name: "tomate", Unit: "KG", AvgUnitCents: 800, SpendCents: 1600},
		{Month: "2026-01", Name: "tomate", Unit: "UN", AvgUnitCents: 150, SpendCents: 300},
		{Month: "2026-02", Name: "tomate", Unit: "KG", AvgUnitCents: 1000, SpendCents: 2000},
		// tomate/UN some no mês 2 → não casa.
	}
	inf := buildInflation(rows)
	if len(inf.Points) != 2 {
		t.Fatalf("esperava 2 pontos, veio %d", len(inf.Points))
	}
	if inf.Points[1].MatchedProducts != 1 {
		t.Fatalf("só (tomate,KG) deveria casar; veio %d", inf.Points[1].MatchedProducts)
	}
	// fator = preço_kg mês2 / mês1 = 1000/800 = 1.25 → índice 125.
	if diff := inf.Points[1].Index - 125.0; diff > 0.01 || diff < -0.01 {
		t.Fatalf("índice esperado ≈125, veio %.4f", inf.Points[1].Index)
	}
}

func TestBuildInflationEmpty(t *testing.T) {
	inf := buildInflation(nil)
	if len(inf.Points) != 0 {
		t.Fatalf("esperava 0 pontos, veio %d", len(inf.Points))
	}
	if inf.Methodology == "" {
		t.Fatal("methodology deve ser preenchida mesmo sem dados")
	}
}

func TestBuildInflationNoMatchKeepsIndex(t *testing.T) {
	// Produtos diferentes a cada mês: sem casamento, índice se mantém (fator 1).
	rows := []MonthlyProductPrice{
		{Month: "2026-01", Name: "picanha", AvgUnitCents: 5000, SpendCents: 10000},
		{Month: "2026-02", Name: "frango", AvgUnitCents: 2000, SpendCents: 4000},
	}
	inf := buildInflation(rows)
	if inf.Points[1].Index != 100 {
		t.Fatalf("sem casamento o índice deve manter 100, veio %v", inf.Points[1].Index)
	}
	if inf.Points[1].MatchedProducts != 0 {
		t.Fatalf("esperava 0 casados, veio %d", inf.Points[1].MatchedProducts)
	}
}

func TestFillMonthlySpendFillsGaps(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []FiscalMonthSpend{{Month: "2026-02", TotalCents: 500, Items: 3}}
	out := fillMonthlySpend(rows, since, 3)
	if len(out) != 3 {
		t.Fatalf("esperava 3 buckets, veio %d", len(out))
	}
	if out[0].Month != "2026-01" || out[0].TotalCents != 0 {
		t.Fatalf("bucket 0 deveria ser 2026-01 zerado, veio %+v", out[0])
	}
	if out[1].Month != "2026-02" || out[1].TotalCents != 500 {
		t.Fatalf("bucket 1 deveria trazer o gasto real, veio %+v", out[1])
	}
	if out[2].Month != "2026-03" {
		t.Fatalf("bucket 2 deveria ser 2026-03, veio %+v", out[2])
	}
}
