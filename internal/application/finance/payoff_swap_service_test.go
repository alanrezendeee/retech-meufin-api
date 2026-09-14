package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// linkGroupToAsset vincula todas as parcelas do grupo ao bem.
func linkGroupToAsset(repo *fakeEntryRepo, groupID, assetID uuid.UUID) {
	t := dom.AssetVehicle
	for _, e := range repo.entries {
		if e.RecurrenceGroupID != nil && *e.RecurrenceGroupID == groupID {
			at, id := t, assetID
			e.AssetType = &at
			e.AssetID = &id
		}
	}
}

// Quitação com desconto paga por terceiro: 48x R$1.000, 20 pagas. Saldo
// R$28.000, quitado por R$24.000. A economia de R$4.000 vira desconto no
// lançamento de quitação e a dívida fecha: pago == total corrigido.
func TestPayoffQuitaComDescontoETerceiro(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 48, 20, 0, 100_000)
	renegs := newFakeRenegRepo(repo)
	svc := NewRenegotiationService(repo, renegs)

	res, err := svc.Payoff(context.Background(), PayoffInput{
		WorkspaceID: ws,
		GroupID:     groupID,
		PayoffCents: 2_400_000,
		Payer:       dom.PayerThirdParty,
	})
	if err != nil {
		t.Fatalf("Payoff: %v", err)
	}
	ev := res.Event
	if ev.Kind != dom.KindPayoff {
		t.Errorf("kind = %s, quer quitacao", ev.Kind)
	}
	if ev.SettledAmountCents != 2_800_000 {
		t.Errorf("settled = %d, quer 2800000", ev.SettledAmountCents)
	}
	if ev.AdjustmentCents != -400_000 {
		t.Errorf("adjustment = %d, quer -400000 (desconto)", ev.AdjustmentCents)
	}
	if ev.OriginCount != 28 {
		t.Errorf("origin_count = %d, quer 28", ev.OriginCount)
	}
	if ev.NewGroupID != nil || ev.NewCount != 0 {
		t.Errorf("quitação não cria série nova")
	}

	pe := res.PayoffEntry
	if pe.Status != dom.StatusRealizada {
		t.Errorf("quitação deve nascer realizada")
	}
	if pe.AmountCents != 2_800_000 || pe.PaidAmountCents == nil || *pe.PaidAmountCents != 2_400_000 {
		t.Errorf("quitação amount=%d paid=%v, quer 2800000/2400000", pe.AmountCents, pe.PaidAmountCents)
	}
	if pe.DiscountCents == nil || *pe.DiscountCents != 400_000 || pe.DiscountReason == nil || *pe.DiscountReason != "quitacao_antecipada" {
		t.Errorf("desconto = %v/%v, quer 400000/quitacao_antecipada", pe.DiscountCents, pe.DiscountReason)
	}
	if pe.PaymentMethod == nil || *pe.PaymentMethod != dom.PaymentCompensacao {
		t.Errorf("terceiro paga por compensação, veio %v", pe.PaymentMethod)
	}
	if pe.RenegotiationID == nil || *pe.RenegotiationID != ev.ID {
		t.Errorf("quitação deve apontar para o evento")
	}
	if ev.PayoffEntryID == nil || *ev.PayoffEntryID != pe.ID {
		t.Errorf("evento deve apontar para a quitação")
	}

	// Origens: canceladas com motivo quitacao e vínculo settled_by.
	cancelled := 0
	for _, e := range repo.entries {
		if e.RecurrenceGroupID != nil && *e.RecurrenceGroupID == groupID && e.Status == dom.StatusCancelada {
			cancelled++
			if e.CancelReason == nil || *e.CancelReason != dom.CancelReasonPayoff {
				t.Errorf("motivo = %v, quer quitacao", e.CancelReason)
			}
			if e.SettledByRenegotiationID == nil || *e.SettledByRenegotiationID != ev.ID {
				t.Errorf("origem sem vínculo settled_by")
			}
		}
	}
	if cancelled != 28 {
		t.Errorf("canceladas = %d, quer 28", cancelled)
	}

	// Linhagem fecha as contas: pago (20 parcelas + quitação) == total corrigido.
	l, err := svc.Lineage(context.Background(), ws, groupID)
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if !l.Settled {
		t.Errorf("dívida quitada deve constar como settled")
	}
	if l.ClosedBy == nil || l.ClosedBy.ID != ev.ID {
		t.Errorf("closed_by ausente ou errado")
	}
	if l.DiscountCents != 400_000 {
		t.Errorf("discount = %d, quer 400000", l.DiscountCents)
	}
	wantPaid := int64(20*100_000 + 2_400_000)
	if l.PaidCents != wantPaid {
		t.Errorf("paid = %d, quer %d", l.PaidCents, wantPaid)
	}
	if l.CurrentTotalCents != wantPaid {
		t.Errorf("total corrigido = %d, quer %d (== pago)", l.CurrentTotalCents, wantPaid)
	}
	if len(l.Stages) != 1 || l.Stages[0].PayoffEntry == nil {
		t.Errorf("etapa deve carregar a quitação")
	}
	if l.Stages[0].CarriedCents != 2_800_000 {
		t.Errorf("carried = %d, quer 2800000", l.Stages[0].CarriedCents)
	}
}

func TestPayoffProprioExigeFormaDePagamento(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 2, 0, 100_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	_, err := svc.Payoff(context.Background(), PayoffInput{
		WorkspaceID: ws, GroupID: groupID, PayoffCents: 900_000, Payer: dom.PayerSelf,
	})
	if err == nil {
		t.Fatalf("quitação própria sem forma de pagamento deveria falhar")
	}

	pix := dom.PaymentPix
	res, err := svc.Payoff(context.Background(), PayoffInput{
		WorkspaceID: ws, GroupID: groupID, PayoffCents: 1_050_000, Payer: dom.PayerSelf, PaymentMethod: &pix,
	})
	if err != nil {
		t.Fatalf("Payoff: %v", err)
	}
	// Pagou acima do saldo (multa): sem desconto, encargo positivo.
	if res.Event.AdjustmentCents != 50_000 {
		t.Errorf("adjustment = %d, quer +50000", res.Event.AdjustmentCents)
	}
	if res.PayoffEntry.DiscountCents != nil {
		t.Errorf("sem desconto quando pagou acima do saldo")
	}
	if *res.PayoffEntry.PaymentMethod != dom.PaymentPix {
		t.Errorf("forma = %v, quer pix", res.PayoffEntry.PaymentMethod)
	}
}

// O caso que motivou a feature: carro financiado dado como entrada num
// carro novo. Concessionária quita o contrato antigo (R$32k sobre saldo de
// R$38.4k), o usado vale R$45k, mais R$5k em pix; contrato novo 48x R$1.890.
func TestAssetSwapCompleto(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, oldGroup := seedInstallmentDebt(repo, 48, 20, 0, 137_143) // 28 abertas ≈ R$38.4k
	oldCar, newCar := uuid.New(), uuid.New()
	linkGroupToAsset(repo, oldGroup, oldCar)
	renegs := newFakeRenegRepo(repo)
	svc := NewRenegotiationService(repo, renegs)

	pix := dom.PaymentPix
	price := int64(9_000_000)
	res, err := svc.AssetSwap(context.Background(), AssetSwapInput{
		WorkspaceID:          ws,
		AssetType:            dom.AssetVehicle,
		OldAssetID:           oldCar,
		NewAssetID:           newCar,
		OldAssetLabel:        "Gol 2018",
		NewAssetLabel:        "Corolla 2025",
		OldGroupID:           &oldGroup,
		PayoffCents:          3_200_000,
		TradeInCents:         4_500_000,
		CashDownpaymentCents: 500_000,
		CashPaymentMethod:    &pix,
		NewAssetPriceCents:   &price,
		NewContract: &NewContractSpec{
			InstallmentCount: 48,
			InstallmentCents: 189_000,
			FirstDueDate:     time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("AssetSwap: %v", err)
	}
	ev := res.Event
	if ev.Kind != dom.KindAssetSwap {
		t.Errorf("kind = %s", ev.Kind)
	}
	wantSettled := int64(28 * 137_143)
	if ev.SettledAmountCents != wantSettled {
		t.Errorf("settled = %d, quer %d", ev.SettledAmountCents, wantSettled)
	}
	if ev.AdjustmentCents != 3_200_000-wantSettled {
		t.Errorf("adjustment = %d", ev.AdjustmentCents)
	}
	if ev.Payer == nil || *ev.Payer != dom.PayerThirdParty {
		t.Errorf("payer default deve ser terceiro")
	}
	if ev.NewGroupID == nil || ev.NewCount != 48 || ev.NewAmountCents != 48*189_000 {
		t.Errorf("série nova: group=%v count=%d amount=%d", ev.NewGroupID, ev.NewCount, ev.NewAmountCents)
	}
	if ev.NetDownpaymentCents() != 4_500_000-3_200_000+500_000 {
		t.Errorf("entrada líquida = %d", ev.NetDownpaymentCents())
	}
	if res.PayoffEntry == nil || res.TradeInEntry == nil || res.DownpaymentEntry == nil {
		t.Fatalf("faltou lançamento: payoff=%v trade=%v cash=%v", res.PayoffEntry, res.TradeInEntry, res.DownpaymentEntry)
	}
	if ev.PayoffEntryID == nil || *ev.PayoffEntryID != res.PayoffEntry.ID ||
		ev.TradeInEntryID == nil || *ev.TradeInEntryID != res.TradeInEntry.ID ||
		ev.DownpaymentEntryID == nil || *ev.DownpaymentEntryID != res.DownpaymentEntry.ID {
		t.Errorf("evento deve apontar para os três lançamentos")
	}
	// Quitação: compensada, bem antigo, com desconto.
	pe := res.PayoffEntry
	if *pe.PaymentMethod != dom.PaymentCompensacao || *pe.AssetID != oldCar {
		t.Errorf("quitação: método=%v asset=%v", pe.PaymentMethod, pe.AssetID)
	}
	if pe.DiscountCents == nil || *pe.DiscountCents != wantSettled-3_200_000 {
		t.Errorf("desconto da quitação = %v", pe.DiscountCents)
	}
	// Venda: receita realizada por compensação, bem antigo.
	te := res.TradeInEntry
	if te.Kind != dom.KindCredit || *te.Type != dom.IncomeTypeAssetSale || te.AmountCents != 4_500_000 ||
		*te.PaymentMethod != dom.PaymentCompensacao || *te.AssetID != oldCar {
		t.Errorf("venda: %+v", te)
	}
	// Entrada: despesa realizada com caixa, bem novo, categoria herdada.
	ce := res.DownpaymentEntry
	if ce.Kind != dom.KindDebit || ce.AmountCents != 500_000 || *ce.PaymentMethod != dom.PaymentPix || *ce.AssetID != newCar {
		t.Errorf("entrada: %+v", ce)
	}
	// Parcelas novas: 48, previstas, bem novo, vínculo ao evento.
	newInst := 0
	for i := range res.Created {
		e := &res.Created[i]
		if e.InstallmentNumber == nil {
			continue
		}
		newInst++
		if e.Status != dom.StatusPrevista || e.AssetID == nil || *e.AssetID != newCar || e.RenegotiationID == nil || *e.RenegotiationID != ev.ID {
			t.Errorf("parcela nova inconsistente: %+v", e)
		}
	}
	if newInst != 48 {
		t.Errorf("parcelas novas = %d", newInst)
	}
	// Efeitos no cadastro do bem.
	if len(renegs.sales) != 1 || renegs.sales[0].AssetID != oldCar || renegs.sales[0].PriceCents != 4_500_000 {
		t.Errorf("venda do bem não registrada: %+v", renegs.sales)
	}
	if len(renegs.acquisitions) != 1 || renegs.acquisitions[0].AssetID != newCar || *renegs.acquisitions[0].PriceCents != price {
		t.Errorf("aquisição do bem não registrada: %+v", renegs.acquisitions)
	}

	// Linhagem do contrato antigo: encerrada pela troca, aponta o sucessor.
	old, err := svc.Lineage(context.Background(), ws, oldGroup)
	if err != nil {
		t.Fatalf("Lineage(old): %v", err)
	}
	if !old.Settled || old.ClosedBy == nil || old.ClosedBy.Kind != dom.KindAssetSwap {
		t.Errorf("contrato antigo deveria estar encerrado pela troca")
	}
	if old.SuccessorGroupID == nil || *old.SuccessorGroupID != *ev.NewGroupID {
		t.Errorf("sucessor = %v, quer %v", old.SuccessorGroupID, ev.NewGroupID)
	}
	if old.PaidCents != int64(20*137_143)+3_200_000 || old.CurrentTotalCents != old.PaidCents {
		t.Errorf("balanço do antigo: paid=%d total=%d", old.PaidCents, old.CurrentTotalCents)
	}

	// Linhagem do contrato novo: é OUTRA dívida (raiz própria), nascida da troca.
	nw, err := svc.Lineage(context.Background(), ws, *ev.NewGroupID)
	if err != nil {
		t.Fatalf("Lineage(new): %v", err)
	}
	if nw.RootGroupID != *ev.NewGroupID {
		t.Errorf("contrato novo deve ser raiz da própria linhagem")
	}
	if nw.OriginEvent == nil || nw.OriginEvent.ID != ev.ID || nw.PredecessorGroupID == nil || *nw.PredecessorGroupID != oldGroup {
		t.Errorf("origem da troca não vinculada: %+v / %v", nw.OriginEvent, nw.PredecessorGroupID)
	}
	if nw.OriginalCents != 48*189_000 || nw.OpenCount != 48 || nw.Settled {
		t.Errorf("balanço do novo: original=%d open=%d settled=%v", nw.OriginalCents, nw.OpenCount, nw.Settled)
	}
	if nw.DiscountCents != 0 || nw.InterestCents != 0 {
		t.Errorf("o desconto da quitação pertence à dívida antiga, não à nova")
	}

	// Por bem: o carro novo tem 1 contrato e 1 evento; o antigo, 1 contrato quitado e o mesmo evento.
	nb, err := svc.AssetDebts(context.Background(), ws, dom.AssetVehicle, newCar)
	if err != nil {
		t.Fatalf("AssetDebts(new): %v", err)
	}
	if len(nb.Debts) != 1 || len(nb.Events) != 1 {
		t.Errorf("bem novo: debts=%d events=%d", len(nb.Debts), len(nb.Events))
	}
	ob, err := svc.AssetDebts(context.Background(), ws, dom.AssetVehicle, oldCar)
	if err != nil {
		t.Fatalf("AssetDebts(old): %v", err)
	}
	if len(ob.Debts) != 1 || !ob.Debts[0].Settled || len(ob.Events) != 1 {
		t.Errorf("bem antigo: debts=%d events=%d", len(ob.Debts), len(ob.Events))
	}
}

// Bem já quitado dado na troca: sem contrato antigo, só venda + contrato novo.
func TestAssetSwapSemContratoAntigo(t *testing.T) {
	repo := newFakeEntryRepo()
	ws := uuid.New()
	oldCar, newCar := uuid.New(), uuid.New()
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	res, err := svc.AssetSwap(context.Background(), AssetSwapInput{
		WorkspaceID:  ws,
		AssetType:    dom.AssetVehicle,
		OldAssetID:   oldCar,
		NewAssetID:   newCar,
		TradeInCents: 3_000_000,
		NewContract: &NewContractSpec{
			InstallmentCount: 24,
			InstallmentCents: 150_000,
			FirstDueDate:     time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("AssetSwap: %v", err)
	}
	if res.PayoffEntry != nil || res.Event.PayoffCents != nil || res.Event.OriginCount != 0 {
		t.Errorf("sem contrato antigo não há quitação")
	}
	if res.TradeInEntry == nil || res.Event.NetDownpaymentCents() != 3_000_000 {
		t.Errorf("venda do usado deveria valer a entrada inteira")
	}
	if len(res.Created) != 25 {
		t.Errorf("created = %d, quer 25 (venda + 24 parcelas)", len(res.Created))
	}
}

func TestAssetSwapRejeitaMesmoBemEEntradaSemForma(t *testing.T) {
	repo := newFakeEntryRepo()
	ws := uuid.New()
	car := uuid.New()
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	if _, err := svc.AssetSwap(context.Background(), AssetSwapInput{
		WorkspaceID: ws, AssetType: dom.AssetVehicle, OldAssetID: car, NewAssetID: car, TradeInCents: 1,
	}); err == nil {
		t.Errorf("mesmo bem dos dois lados deveria falhar")
	}
	if _, err := svc.AssetSwap(context.Background(), AssetSwapInput{
		WorkspaceID: ws, AssetType: dom.AssetVehicle, OldAssetID: car, NewAssetID: uuid.New(),
		TradeInCents: 1_000_000, CashDownpaymentCents: 100_000,
	}); err == nil {
		t.Errorf("entrada em dinheiro sem forma de pagamento deveria falhar")
	}
}

// A renegociação comum segue encadeando a linhagem; a quitação depois dela
// fecha a cadeia inteira: raiz → acordo → quitação.
func TestLineageRenegociacaoDepoisQuitacao(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, root := seedInstallmentDebt(repo, 12, 4, 0, 100_000)
	svc := NewRenegotiationService(repo, newFakeRenegRepo(repo))

	rn, err := svc.Renegotiate(context.Background(), RenegotiateInput{
		WorkspaceID: ws, GroupID: root, InstallmentCount: 10, InstallmentCents: 90_000,
		FirstDueDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Renegotiate: %v", err)
	}
	if _, err := svc.Payoff(context.Background(), PayoffInput{
		WorkspaceID: ws, GroupID: *rn.Renegotiation.NewGroupID, PayoffCents: 800_000, Payer: dom.PayerThirdParty,
	}); err != nil {
		t.Fatalf("Payoff: %v", err)
	}
	l, err := svc.Lineage(context.Background(), ws, *rn.Renegotiation.NewGroupID)
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if l.RootGroupID != root || len(l.Stages) != 2 {
		t.Fatalf("root=%v stages=%d", l.RootGroupID, len(l.Stages))
	}
	if !l.Settled || l.ClosedBy == nil || l.ClosedBy.Kind != dom.KindPayoff {
		t.Errorf("linhagem deveria terminar na quitação")
	}
	// original 1.2M; acordo 900k sobre 800k (+100k); quitação 800k sobre 900k (−100k).
	if l.InterestCents != 100_000 || l.DiscountCents != 100_000 {
		t.Errorf("juros=%d desconto=%d", l.InterestCents, l.DiscountCents)
	}
	if l.PaidCents != 400_000+800_000 || l.CurrentTotalCents != 1_200_000 {
		t.Errorf("paid=%d total=%d", l.PaidCents, l.CurrentTotalCents)
	}
}
