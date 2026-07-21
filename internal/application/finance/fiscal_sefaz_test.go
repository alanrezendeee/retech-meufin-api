package finance

import (
	"encoding/json"
	"testing"

	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/infosimples"
)

// TestFiscalJSONFromNFCe_RoundTrip garante que o resultado da SEFAZ, convertido
// para o schema fiscal-extract-v1 e relido por ParseFiscal, preserva os valores
// em centavos e a quantidade em milésimos (o round-trip centavos→reais→centavos
// não pode perder precisão).
func TestFiscalJSONFromNFCe_RoundTrip(t *testing.T) {
	r := &infosimples.NFCeResult{
		EmitenteNome:    "SUPERMERCADOS ARCHER SA",
		EmitenteCNPJ:    "79.257.291/0001-58",
		DataEmissao:     "2026-07-16",
		ValorTotalCents: 14938, // soma dos 2 itens do fixture (1940 + 12998)
		Produtos: []infosimples.NFCeProduto{
			{Descricao: "LINGUICA OLHO BLUMENAU", QuantityMilli: 290, UnitCents: 6690, AmountCents: 1940, Codigo: "111"},
			{Descricao: "CAFE ORFEU GRAO 1KG", QuantityMilli: 1000, UnitCents: 12998, AmountCents: 12998, Codigo: "222"},
		},
	}

	js, err := json.Marshal(storedFiscalFromNFCe(r))
	if err != nil {
		t.Fatalf("marshal storedFiscalFromNFCe: %v", err)
	}

	svc := &FinanceExtractionService{}
	sug, err := svc.ParseFiscal(&dom.FinanceDocument{ExtractedJSON: js})
	if err != nil {
		t.Fatalf("ParseFiscal: %v", err)
	}

	if sug.Merchant != r.EmitenteNome {
		t.Fatalf("merchant: got %q, want %q", sug.Merchant, r.EmitenteNome)
	}
	if sug.CNPJ != r.EmitenteCNPJ {
		t.Fatalf("cnpj: got %q, want %q", sug.CNPJ, r.EmitenteCNPJ)
	}
	if sug.Date != "2026-07-16" {
		t.Fatalf("date: got %q, want 2026-07-16", sug.Date)
	}
	if sug.TotalCents != 14938 {
		t.Fatalf("total_cents: got %d, want 14938", sug.TotalCents)
	}
	if len(sug.Items) != 2 {
		t.Fatalf("items: got %d, want 2", len(sug.Items))
	}

	i0 := sug.Items[0]
	if i0.AmountCents != 1940 || i0.UnitCents != 6690 || i0.QuantityMilli != 290 {
		t.Fatalf("item[0] valores errados: amount=%d unit=%d qty=%d", i0.AmountCents, i0.UnitCents, i0.QuantityMilli)
	}
	i1 := sug.Items[1]
	if i1.AmountCents != 12998 || i1.UnitCents != 12998 || i1.QuantityMilli != 1000 {
		t.Fatalf("item[1] valores errados: amount=%d unit=%d qty=%d", i1.AmountCents, i1.UnitCents, i1.QuantityMilli)
	}

	// A soma dos itens tem de fechar com o total da nota (reconciliação).
	var soma int64
	for _, it := range sug.Items {
		soma += it.AmountCents
	}
	if soma != sug.TotalCents {
		t.Fatalf("soma dos itens (%d) != total (%d)", soma, sug.TotalCents)
	}
}
