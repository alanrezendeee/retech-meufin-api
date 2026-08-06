package finance

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// fakeRenegRepo implementa dom.RenegotiationRepository em memória.
type fakeRenegRepo struct {
	entries *fakeEntryRepo
	events  map[uuid.UUID]*dom.Renegotiation
	// failApply simula erro no meio da transação.
	failApply error
}

func newFakeRenegRepo(entries *fakeEntryRepo) *fakeRenegRepo {
	return &fakeRenegRepo{entries: entries, events: map[uuid.UUID]*dom.Renegotiation{}}
}

func (f *fakeRenegRepo) Apply(_ context.Context, r *dom.Renegotiation, originIDs []uuid.UUID, newEntries []*dom.FinancialEntry) error {
	if f.failApply != nil {
		return f.failApply
	}
	f.events[r.ID] = r
	for _, id := range originIDs {
		e, ok := f.entries.entries[id]
		if !ok || e.Status != dom.StatusPrevista {
			return &dom.ValidationError{Msg: "as cobranças mudaram durante a renegociação"}
		}
		e.Status = dom.StatusCancelada
		reason := dom.CancelReasonRenegotiation
		e.CancelReason = &reason
		e.RenegotiationID = &r.ID
	}
	for _, e := range newEntries {
		cp := *e
		f.entries.entries[e.ID] = &cp
	}
	return nil
}

func (f *fakeRenegRepo) GetByID(_ context.Context, workspaceID, id uuid.UUID) (*dom.Renegotiation, error) {
	r, ok := f.events[id]
	if !ok || r.WorkspaceID != workspaceID {
		return nil, dom.ErrNotFound
	}
	return r, nil
}

func (f *fakeRenegRepo) List(_ context.Context, workspaceID uuid.UUID, _, _ int) ([]dom.Renegotiation, int64, error) {
	var out []dom.Renegotiation
	for _, r := range f.events {
		if r.WorkspaceID == workspaceID {
			out = append(out, *r)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeRenegRepo) ListEntries(_ context.Context, workspaceID, renegotiationID uuid.UUID) ([]dom.FinancialEntry, []dom.FinancialEntry, error) {
	var origins, created []dom.FinancialEntry
	for _, e := range f.entries.entries {
		if e.WorkspaceID != workspaceID || e.RenegotiationID == nil || *e.RenegotiationID != renegotiationID {
			continue
		}
		if e.Status == dom.StatusCancelada {
			origins = append(origins, *e)
		} else {
			created = append(created, *e)
		}
	}
	sort.Slice(created, func(i, j int) bool { return created[i].DueDate.Before(created[j].DueDate) })
	return origins, created, nil
}

// seedInstallmentDebt monta o cenário real: parcelamento de `total` parcelas de
// `amount`, com `paid` já liquidadas, das quais as últimas `partial` foram
// pagas parcialmente (metade), cada uma gerando um residual em aberto.
func seedInstallmentDebt(repo *fakeEntryRepo, total, paid, partial int, amount int64) (uuid.UUID, uuid.UUID) {
	ws := uuid.New()
	groupID := uuid.New()
	start := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)

	for i := 1; i <= total; i++ {
		num, tot := i, total
		e := &dom.FinancialEntry{
			ID:                uuid.New(),
			WorkspaceID:       ws,
			Kind:              dom.KindDebit,
			Status:            dom.StatusPrevista,
			AmountCents:       amount,
			DueDate:           start.AddDate(0, i-1, 0),
			Description:       "financiamento carro",
			Recurrence:        dom.RecurrenceNone,
			RecurrenceGroupID: &groupID,
			InstallmentNumber: &num,
			InstallmentTotal:  &tot,
		}
		if i <= paid {
			e.Status = dom.StatusRealizada
			full := amount
			e.PaidAmountCents = &full
			t := e.DueDate
			e.PaidAt = &t
			// As últimas `partial` pagas foram parciais: metade paga, metade
			// virou residual em aberto.
			if i > paid-partial {
				half := amount / 2
				e.PaidAmountCents = &half
				res := &dom.FinancialEntry{
					ID:           uuid.New(),
					WorkspaceID:  ws,
					Kind:         dom.KindDebit,
					Status:       dom.StatusPrevista,
					AmountCents:  amount - half,
					DueDate:      e.DueDate,
					Description:  "Residual de " + e.Description,
					Recurrence:   dom.RecurrenceNone,
					ResidualOfID: &e.ID,
				}
				repo.entries[res.ID] = res
			}
		}
		repo.entries[e.ID] = e
	}
	return ws, groupID
}

// O caso concreto: 100x R$500, 39 pagas, as 10 últimas parciais (R$250 cada).
// Saldo = 61 parcelas cheias + 10 residuais de R$250.
func TestPreviewApuraSaldoComResiduais(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 100, 39, 10, 50_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	p, err := svc.PreviewGroup(context.Background(), ws, groupID)
	if err != nil {
		t.Fatalf("PreviewGroup: %v", err)
	}
	if p.InstallmentCount != 61 {
		t.Errorf("parcelas em aberto = %d, quer 61", p.InstallmentCount)
	}
	if p.ResidualCount != 10 {
		t.Errorf("residuais = %d, quer 10", p.ResidualCount)
	}
	wantOpen := int64(61*50_000 + 10*25_000)
	if p.OpenTotalCents != wantOpen {
		t.Errorf("saldo = %d, quer %d", p.OpenTotalCents, wantOpen)
	}
	if p.PaidCount != 39 {
		t.Errorf("pagas = %d, quer 39", p.PaidCount)
	}
	// Pagas: 29 cheias + 10 pela metade.
	wantPaid := int64(29*50_000 + 10*25_000)
	if p.PaidCents != wantPaid {
		t.Errorf("pago = %d, quer %d", p.PaidCents, wantPaid)
	}
	if len(p.Charges) != 71 {
		t.Errorf("cobranças = %d, quer 71", len(p.Charges))
	}
}

// A armadilha central: a parcela paga parcialmente NÃO pode entrar pelo valor
// cheio, senão a mesma dívida é contada duas vezes (parcela + residual).
func TestPreviewNaoContaParcelaParcialEmDobro(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 10, 10, 10, 50_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	p, err := svc.PreviewGroup(context.Background(), ws, groupID)
	if err != nil {
		t.Fatalf("PreviewGroup: %v", err)
	}
	if p.InstallmentCount != 0 {
		t.Errorf("nenhuma parcela prevista deveria entrar, veio %d", p.InstallmentCount)
	}
	// Só os 10 residuais de R$250 — não os R$500 das parcelas já liquidadas.
	if p.OpenTotalCents != 10*25_000 {
		t.Errorf("saldo = %d, quer %d (só os residuais)", p.OpenTotalCents, 10*25_000)
	}
}

func TestRenegotiateEncerraOrigensECriaSerie(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 100, 39, 10, 50_000)
	renegRepo := newFakeRenegRepo(repo)
	svc := NewRenegotiationService(repo, renegRepo)

	res, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID:      ws,
		GroupID:          groupID,
		InstallmentCount: 80,
		InstallmentCents: 40_000,
		FirstDueDate:     time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Renegotiate: %v", err)
	}

	r := res.Renegotiation
	wantSettled := int64(61*50_000 + 10*25_000)
	if r.SettledAmountCents != wantSettled {
		t.Errorf("saldo apurado = %d, quer %d", r.SettledAmountCents, wantSettled)
	}
	if r.NewAmountCents != 80*40_000 {
		t.Errorf("novo total = %d, quer %d", r.NewAmountCents, 80*40_000)
	}
	if r.AdjustmentCents != r.NewAmountCents-wantSettled {
		t.Errorf("ajuste = %d, quer %d", r.AdjustmentCents, r.NewAmountCents-wantSettled)
	}
	if r.OriginCount != 71 || r.NewCount != 80 {
		t.Errorf("contagens = %d origens / %d novas, quer 71/80", r.OriginCount, r.NewCount)
	}
	if len(res.Created) != 80 {
		t.Fatalf("criadas = %d, quer 80", len(res.Created))
	}

	// Todas as novas vinculadas ao evento, previstas e numeradas.
	for i := range res.Created {
		e := res.Created[i]
		if e.RenegotiationID == nil || *e.RenegotiationID != r.ID {
			t.Fatalf("parcela nova sem vínculo com o evento")
		}
		if e.Status != dom.StatusPrevista {
			t.Fatalf("parcela nova = %s, quer prevista", e.Status)
		}
		if e.InstallmentTotal == nil || *e.InstallmentTotal != 80 {
			t.Fatalf("installment_total = %v, quer 80", e.InstallmentTotal)
		}
	}

	// Origens encerradas com motivo renegociacao e vínculo.
	origins, created, err := renegRepo.ListEntries(context.Background(), ws, r.ID)
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(origins) != 71 || len(created) != 80 {
		t.Fatalf("evento liga %d origens e %d novas, quer 71/80", len(origins), len(created))
	}
	for _, o := range origins {
		if o.Status != dom.StatusCancelada {
			t.Fatalf("origem = %s, quer cancelada", o.Status)
		}
		if o.CancelReason == nil || *o.CancelReason != dom.CancelReasonRenegotiation {
			t.Fatalf("origem sem motivo renegociacao: %v", o.CancelReason)
		}
	}

	// As 39 pagas permanecem intactas — fato consumado.
	paidUntouched := 0
	for _, e := range repo.entries {
		if e.Status == dom.StatusRealizada && e.RenegotiationID == nil {
			paidUntouched++
		}
	}
	if paidUntouched != 39 {
		t.Errorf("parcelas pagas intactas = %d, quer 39", paidUntouched)
	}
}

func TestRenegotiateDescontoEEncargo(t *testing.T) {
	cases := []struct {
		name      string
		count     int
		cents     int64
		wantSign  int
		wantLabel string
	}{
		// Saldo do cenário: 61×R$500 + 10×R$250 = R$33.000.
		{"com encargo", 80, 45_000, 1, "juros"},      // 80×R$450 = R$36.000
		{"com desconto", 80, 40_000, -1, "desconto"}, // 80×R$400 = R$32.000
	}
	for _, tc := range cases {
		repo := newFakeEntryRepo()
		ws, groupID := seedInstallmentDebt(repo, 100, 39, 10, 50_000)
		svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

		res, err := svc.Renegotiate(context.Background(), RenegotiateInput{
			WorkspaceID: ws, GroupID: groupID,
			InstallmentCount: tc.count, InstallmentCents: tc.cents,
			FirstDueDate: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		adj := res.Renegotiation.AdjustmentCents
		if tc.wantSign > 0 && adj <= 0 {
			t.Errorf("%s: ajuste = %d, queria positivo (%s)", tc.name, adj, tc.wantLabel)
		}
		if tc.wantSign < 0 && adj >= 0 {
			t.Errorf("%s: ajuste = %d, queria negativo (%s)", tc.name, adj, tc.wantLabel)
		}
		if res.Renegotiation.IsDiscount() != (tc.wantSign < 0) {
			t.Errorf("%s: IsDiscount inconsistente", tc.name)
		}
	}
}

func TestRenegotiateRejeitaDividaQuitada(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 10, 10, 0, 50_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	_, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID: ws, GroupID: groupID,
		InstallmentCount: 5, InstallmentCents: 10_000,
		FirstDueDate: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("dívida quitada não deveria poder ser renegociada")
	}
}

func TestRenegotiateValidaEntrada(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 100, 39, 10, 50_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))
	due := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		in   RenegotiateInput
	}{
		{"sem parcelas", RenegotiateInput{WorkspaceID: ws, GroupID: groupID, InstallmentCount: 0, InstallmentCents: 40_000, FirstDueDate: due}},
		{"valor zero", RenegotiateInput{WorkspaceID: ws, GroupID: groupID, InstallmentCount: 10, InstallmentCents: 0, FirstDueDate: due}},
		{"sem vencimento", RenegotiateInput{WorkspaceID: ws, GroupID: groupID, InstallmentCount: 10, InstallmentCents: 40_000}},
	}
	for _, tc := range cases {
		if _, err := svc.Renegotiate(context.Background(), tc.in); err == nil {
			t.Errorf("%s: deveria ser rejeitado", tc.name)
		}
	}
}

// Se a transação falhar, nada pode ter sido alterado — dívida encerrada sem
// série nova sumiria dos relatórios.
func TestRenegotiateNaoAlteraNadaQuandoFalha(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 100, 39, 10, 50_000)
	renegRepo := newFakeRenegRepo(repo)
	renegRepo.failApply = &dom.ValidationError{Msg: "falha simulada"}
	svc := NewRenegotiationService(repo, renegRepo)

	before := len(repo.entries)
	_, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID: ws, GroupID: groupID,
		InstallmentCount: 80, InstallmentCents: 40_000,
		FirstDueDate: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("erro na aplicação deveria propagar")
	}
	if len(repo.entries) != before {
		t.Errorf("nenhum lançamento deveria ser criado, foi de %d para %d", before, len(repo.entries))
	}
	for _, e := range repo.entries {
		if e.RenegotiationID != nil {
			t.Fatal("nenhum lançamento deveria ter sido vinculado")
		}
		if e.CancelReason != nil {
			t.Fatal("nenhuma origem deveria ter sido cancelada")
		}
	}
}

// Reabrir uma parcela cujo residual foi renegociado recriaria dívida já
// repactuada.
func TestReopenBloqueadoPorResidualRenegociado(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusRealizada)
	renegID := uuid.New()
	res := &dom.FinancialEntry{
		ID:              uuid.New(),
		WorkspaceID:     e.WorkspaceID,
		Kind:            dom.KindDebit,
		Status:          dom.StatusCancelada,
		AmountCents:     5_000,
		DueDate:         e.DueDate,
		Description:     "Residual de " + e.Description,
		ResidualOfID:    &e.ID,
		RenegotiationID: &renegID,
	}
	repo.entries[res.ID] = res
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID); err == nil {
		t.Fatal("reabertura deveria ser bloqueada quando o residual foi renegociado")
	}
}
