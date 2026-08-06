package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// --- Isenção: cobrança não devida no período ---

func TestWaiveZerosPaidKeepsExpected(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	got, err := svc.Waive(context.Background(), e.WorkspaceID, e.ID, "bonus", nil)
	if err != nil {
		t.Fatalf("Waive: %v", err)
	}
	if got.Status != dom.StatusRealizada {
		t.Fatalf("status = %s, quer realizada", got.Status)
	}
	// O previsto continua valendo o valor cheio: a mensalidade não mudou.
	if got.AmountCents != e.AmountCents {
		t.Fatalf("amount_cents = %d, quer %d (previsto não muda)", got.AmountCents, e.AmountCents)
	}
	if got.PaidAmountCents == nil || *got.PaidAmountCents != 0 {
		t.Fatalf("paid_amount_cents = %v, quer 0", got.PaidAmountCents)
	}
	if got.DiscountCents == nil || *got.DiscountCents != e.AmountCents {
		t.Fatalf("discount_cents = %v, quer %d (integral)", got.DiscountCents, e.AmountCents)
	}
	if got.DiscountReason == nil || *got.DiscountReason != "bonus" {
		t.Fatalf("discount_reason = %v, quer bonus", got.DiscountReason)
	}
	if got.PaidAt == nil {
		t.Fatal("paid_at deve ser preenchido")
	}
}

// A isenção não pode gerar residual: pago 0 com desconto integral significa
// nada a pagar, não "R$ 98,57 em aberto".
func TestWaiveDoesNotCreateResidual(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Waive(context.Background(), e.WorkspaceID, e.ID, "cortesia", nil); err != nil {
		t.Fatalf("Waive: %v", err)
	}
	residuals, err := repo.ListResiduals(context.Background(), e.WorkspaceID, e.ID)
	if err != nil {
		t.Fatalf("ListResiduals: %v", err)
	}
	if len(residuals) != 0 {
		t.Fatalf("isenção não deve gerar residual, veio %d", len(residuals))
	}
}

func TestWaiveRequiresReason(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Waive(context.Background(), e.WorkspaceID, e.ID, "", nil); err == nil {
		t.Fatal("isenção sem motivo deveria ser rejeitada")
	}
}

func TestWaiveRejectsAlreadySettledOrCancelled(t *testing.T) {
	for _, st := range []dom.Status{dom.StatusRealizada, dom.StatusCancelada} {
		repo := newFakeEntryRepo()
		e := seedEntry(repo, st)
		svc := NewFinancialEntryService(repo, fakeCategoryRepo{})
		if _, err := svc.Waive(context.Background(), e.WorkspaceID, e.ID, "bonus", nil); err == nil {
			t.Fatalf("isenção de lançamento %s deveria ser rejeitada", st)
		}
	}
}

// Reabrir uma isenção limpa desconto e valor pago, voltando a prevista.
func TestReopenAfterWaive(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Waive(context.Background(), e.WorkspaceID, e.ID, "ressarcimento", nil); err != nil {
		t.Fatalf("Waive: %v", err)
	}
	got, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if got.Status != dom.StatusPrevista {
		t.Fatalf("status = %s, quer prevista", got.Status)
	}
	if got.DiscountCents != nil || got.DiscountReason != nil || got.PaidAmountCents != nil {
		t.Fatalf("reabertura deve limpar desconto e pagamento, veio %+v", got)
	}
}

// Pago zero SEM desconto integral continua proibido — seria um lançamento
// marcado como pago sem pagamento, e geraria residual do valor cheio.
func TestConfirmRejectsZeroPaidWithoutFullDiscount(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	zero := int64(0)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAmountCents: &zero,
	}); err == nil {
		t.Fatal("pago zero sem desconto integral deveria ser rejeitado")
	}

	// Desconto parcial também não autoriza pago zero.
	half := e.AmountCents / 2
	reason := "promocao"
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID,
		DiscountCents: &half, DiscountReason: &reason, PaidAmountCents: &zero,
	}); err == nil {
		t.Fatal("pago zero com desconto parcial deveria ser rejeitado")
	}
}

func TestDiscountGreaterThanAmountStillInvalid(t *testing.T) {
	reason := "bonus"
	over := int64(10_001)
	e := &dom.FinancialEntry{
		ID: uuid.New(), WorkspaceID: uuid.New(), Kind: dom.KindDebit,
		Status: dom.StatusRealizada, AmountCents: 10_000,
		DueDate: time.Now().UTC(), Description: "conta",
		DiscountCents: &over, DiscountReason: &reason,
	}
	if err := e.Validate(); err == nil {
		t.Fatal("desconto maior que o valor deveria ser rejeitado")
	}
	// Igual ao valor é válido (isenção total).
	equal := int64(10_000)
	e.DiscountCents = &equal
	if err := e.Validate(); err != nil {
		t.Fatalf("desconto igual ao valor deve ser aceito (isenção): %v", err)
	}
}

// --- Cancelamento com motivo e efeito na recorrência ---

func TestCancelStoresReason(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	reason := "sem_cobranca_no_mes"
	got, err := svc.Cancel(context.Background(), e.WorkspaceID, e.ID, &reason)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got.Status != dom.StatusCancelada {
		t.Fatalf("status = %s, quer cancelada", got.Status)
	}
	if got.CancelReason == nil || *got.CancelReason != reason {
		t.Fatalf("cancel_reason = %v, quer %s", got.CancelReason, reason)
	}
}

func TestCancelRejectsUnknownReason(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	bogus := "motivo_inventado"
	if _, err := svc.Cancel(context.Background(), e.WorkspaceID, e.ID, &bogus); err == nil {
		t.Fatal("motivo fora do catálogo deveria ser rejeitado")
	}
}

func TestCancelReasonEndsRecurrenceMapping(t *testing.T) {
	ends := "encerramento"
	keeps := "sem_cobranca_no_mes"
	if !dom.CancelReasonEndsRecurrence(&ends) {
		t.Error("encerramento deve encerrar a série")
	}
	if dom.CancelReasonEndsRecurrence(&keeps) {
		t.Error("sem_cobranca_no_mes NÃO deve encerrar a série")
	}
	// Legado: cancelamento sem motivo preserva o comportamento antigo.
	if !dom.CancelReasonEndsRecurrence(nil) {
		t.Error("cancelamento sem motivo deve encerrar (compatibilidade)")
	}
}

// seedRecurringFrontier cria a ocorrência mais futura de um grupo recorrente,
// com vencimento já dentro da janela que dispara a extensão.
func seedRecurringFrontier(repo *fakeEntryRepo, status dom.Status, reason *string) *dom.FinancialEntry {
	groupID := uuid.New()
	e := &dom.FinancialEntry{
		ID:                uuid.New(),
		WorkspaceID:       uuid.New(),
		Kind:              dom.KindDebit,
		Status:            status,
		AmountCents:       9_857,
		DueDate:           time.Now().UTC().AddDate(0, 2, 0),
		Description:       "energia apto Brusque",
		Recurrence:        dom.RecurrenceMonthly,
		RecurrenceGroupID: &groupID,
		CancelReason:      reason,
	}
	repo.entries[e.ID] = e
	return e
}

// O caso que motivou o motivo de cancelamento: cancelar a ocorrência mais
// futura por "não houve cobrança" matava a série em silêncio.
func TestExtendRecurrencesKeepsSeriesWhenMonthSkipped(t *testing.T) {
	repo := newFakeEntryRepo()
	reason := "sem_cobranca_no_mes"
	e := seedRecurringFrontier(repo, dom.StatusCancelada, &reason)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	created, err := svc.ExtendRecurrences(context.Background())
	if err != nil {
		t.Fatalf("ExtendRecurrences: %v", err)
	}
	if created == 0 {
		t.Fatal("série deveria continuar sendo estendida após cancelar um mês pontual")
	}
	// As ocorrências novas nascem previstas e sem o motivo do cancelamento.
	for _, got := range repo.entries {
		if got.ID == e.ID {
			continue
		}
		if got.Status != dom.StatusPrevista {
			t.Fatalf("ocorrência nova = %s, quer prevista", got.Status)
		}
		if got.CancelReason != nil {
			t.Fatalf("ocorrência nova não deve herdar cancel_reason, veio %v", *got.CancelReason)
		}
	}
}

func TestExtendRecurrencesStopsOnEncerramento(t *testing.T) {
	repo := newFakeEntryRepo()
	reason := "encerramento"
	seedRecurringFrontier(repo, dom.StatusCancelada, &reason)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	created, err := svc.ExtendRecurrences(context.Background())
	if err != nil {
		t.Fatalf("ExtendRecurrences: %v", err)
	}
	if created != 0 {
		t.Fatalf("série encerrada não deve ser estendida, criou %d", created)
	}
}

func TestExtendRecurrencesStopsOnLegacyCancelWithoutReason(t *testing.T) {
	repo := newFakeEntryRepo()
	seedRecurringFrontier(repo, dom.StatusCancelada, nil)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	created, err := svc.ExtendRecurrences(context.Background())
	if err != nil {
		t.Fatalf("ExtendRecurrences: %v", err)
	}
	if created != 0 {
		t.Fatalf("cancelamento legado (sem motivo) deve encerrar a série, criou %d", created)
	}
}
