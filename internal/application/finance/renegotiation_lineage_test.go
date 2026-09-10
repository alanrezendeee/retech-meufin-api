package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// Cenário: 6x100 com 2 pagas → acordo 1 (400 em aberto → 5x100 = 500, +100)
// → 1 parcela do acordo paga → acordo 2 (400 em aberto → 2x180 = 360, −40).
func seedLineage(t *testing.T) (*RenegotiationService, *fakeEntryRepo, uuid.UUID, uuid.UUID, *dom.Renegotiation, *dom.Renegotiation) {
	t.Helper()
	repo := newFakeEntryRepo()
	renegs := newFakeRenegRepo(repo)
	svc := NewRenegotiationService(repo, renegs)
	ws, root := seedInstallmentDebt(repo, 6, 2, 0, 10000)

	r1, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID: ws, GroupID: root, InstallmentCount: 5, InstallmentCents: 10000,
		FirstDueDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("acordo 1: %v", err)
	}
	// Paga a primeira parcela do acordo 1.
	first := repo.entries[r1.Created[0].ID]
	first.Status = dom.StatusRealizada
	full := first.AmountCents
	first.PaidAmountCents = &full

	g1 := *r1.Renegotiation.NewGroupID
	r2, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID: ws, GroupID: g1, InstallmentCount: 2, InstallmentCents: 18000,
		FirstDueDate: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("acordo 2: %v", err)
	}
	return svc, repo, ws, root, r1.Renegotiation, r2.Renegotiation
}

func TestLineageEncadeiaAcordosETotaliza(t *testing.T) {
	svc, _, ws, root, r1, r2 := seedLineage(t)

	// A partir de qualquer grupo da cadeia chega-se à mesma linhagem.
	for _, g := range []uuid.UUID{root, *r1.NewGroupID, *r2.NewGroupID} {
		l, err := svc.Lineage(context.Background(), ws, g)
		if err != nil {
			t.Fatalf("Lineage(%s): %v", g, err)
		}
		if l.RootGroupID != root || l.CurrentGroupID != *r2.NewGroupID {
			t.Fatalf("raiz/vigente errados: %+v", l)
		}
		if len(l.Stages) != 3 {
			t.Fatalf("esperava 3 etapas, veio %d", len(l.Stages))
		}
		if l.OriginalCents != 60000 || l.InterestCents != 10000 || l.DiscountCents != 4000 || l.CurrentTotalCents != 66000 {
			t.Fatalf("balanço errado: original=%d juros=%d desconto=%d total=%d", l.OriginalCents, l.InterestCents, l.DiscountCents, l.CurrentTotalCents)
		}
		if l.PaidCents != 30000 || l.PaidCount != 3 || l.OpenCents != 36000 || l.OpenCount != 2 || l.Settled {
			t.Fatalf("pago/aberto errados: %+v", l)
		}
		if l.RenegotiationCnt != 2 {
			t.Fatalf("esperava 2 acordos, veio %d", l.RenegotiationCnt)
		}
	}

	l, _ := svc.Lineage(context.Background(), ws, root)
	s0, s1, s2 := l.Stages[0], l.Stages[1], l.Stages[2]
	if s0.Renegotiation != nil || s0.SettledBy == nil || s0.SettledBy.ID != r1.ID {
		t.Fatalf("etapa 0 deve ser original encerrada pelo acordo 1: %+v", s0)
	}
	if s0.PaidCount != 2 || s0.PaidCents != 20000 || s0.CarriedCount != 4 || s0.CarriedCents != 40000 || s0.OpenCount != 0 {
		t.Fatalf("etapa 0 errada: %+v", s0)
	}
	if s1.Renegotiation == nil || s1.Renegotiation.ID != r1.ID || s1.SettledBy == nil || s1.SettledBy.ID != r2.ID {
		t.Fatalf("etapa 1 deve ser criada pelo acordo 1 e encerrada pelo 2: %+v", s1)
	}
	if s1.TotalCents != 50000 || s1.PaidCount != 1 || s1.CarriedCount != 4 || s1.CarriedCents != 40000 {
		t.Fatalf("etapa 1 errada: %+v", s1)
	}
	if s2.Renegotiation == nil || s2.Renegotiation.ID != r2.ID || s2.SettledBy != nil {
		t.Fatalf("etapa 2 deve ser a vigente: %+v", s2)
	}
	if s2.TotalCents != 36000 || s2.OpenCount != 2 || s2.OpenCents != 36000 || len(s2.Entries) != 2 {
		t.Fatalf("etapa 2 errada: %+v", s2)
	}
}

func TestGetPreservaCriadasAposNovoAcordoEExpoePagasAntes(t *testing.T) {
	svc, _, ws, root, r1, r2 := seedLineage(t)

	d1, err := svc.Get(context.Background(), ws, r1.ID)
	if err != nil {
		t.Fatalf("Get r1: %v", err)
	}
	// Antes, renegociar de novo sobrescrevia o vínculo e o acordo 1 perdia
	// as parcelas que criou.
	if len(d1.Created) != 5 || len(d1.Origins) != 4 {
		t.Fatalf("acordo 1: esperava 5 criadas e 4 origens, veio %d/%d", len(d1.Created), len(d1.Origins))
	}
	if len(d1.PaidBefore) != 2 || d1.PaidBeforeCents != 20000 {
		t.Fatalf("acordo 1: pagas antes = %d / %d", len(d1.PaidBefore), d1.PaidBeforeCents)
	}
	if d1.PreviousID != nil || d1.NextID == nil || *d1.NextID != r2.ID || d1.RootGroupID == nil || *d1.RootGroupID != root {
		t.Fatalf("acordo 1: vizinhos errados: prev=%v next=%v root=%v", d1.PreviousID, d1.NextID, d1.RootGroupID)
	}

	d2, err := svc.Get(context.Background(), ws, r2.ID)
	if err != nil {
		t.Fatalf("Get r2: %v", err)
	}
	if len(d2.Created) != 2 || len(d2.Origins) != 4 {
		t.Fatalf("acordo 2: esperava 2 criadas e 4 origens, veio %d/%d", len(d2.Created), len(d2.Origins))
	}
	if len(d2.PaidBefore) != 1 || d2.PaidBeforeCents != 10000 {
		t.Fatalf("acordo 2: pagas antes = %d / %d", len(d2.PaidBefore), d2.PaidBeforeCents)
	}
	if d2.PreviousID == nil || *d2.PreviousID != r1.ID || d2.NextID != nil || *d2.RootGroupID != root {
		t.Fatalf("acordo 2: vizinhos errados: prev=%v next=%v", d2.PreviousID, d2.NextID)
	}
}

func TestLineageParcelamentoSemAcordo(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))
	ws, root := seedInstallmentDebt(repo, 4, 1, 0, 25000)

	l, err := svc.Lineage(context.Background(), ws, root)
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if len(l.Stages) != 1 || l.RenegotiationCnt != 0 || l.OriginalCents != 100000 || l.CurrentTotalCents != 100000 {
		t.Fatalf("linhagem simples errada: %+v", l)
	}
	if l.PaidCents != 25000 || l.OpenCents != 75000 || l.Stages[0].SettledBy != nil {
		t.Fatalf("balanço simples errado: %+v", l)
	}
	if _, err := svc.Lineage(context.Background(), ws, uuid.New()); err == nil {
		t.Fatalf("grupo inexistente deveria falhar")
	}
}
