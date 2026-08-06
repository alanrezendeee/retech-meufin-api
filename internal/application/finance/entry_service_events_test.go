package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// fakeEventRepo acumula a trilha em memória.
type fakeEventRepo struct {
	events []dom.EntryEvent
}

func (f *fakeEventRepo) Create(_ context.Context, ev *dom.EntryEvent) error {
	f.events = append(f.events, *ev)
	return nil
}

func (f *fakeEventRepo) ListByEntry(_ context.Context, workspaceID, entryID uuid.UUID) ([]dom.EntryEvent, error) {
	var out []dom.EntryEvent
	for _, ev := range f.events {
		if ev.WorkspaceID == workspaceID && ev.EntryID == entryID {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeEventRepo) last(t *testing.T) dom.EntryEvent {
	t.Helper()
	if len(f.events) == 0 {
		t.Fatal("nenhum evento gravado")
	}
	return f.events[len(f.events)-1]
}

func TestEventsConfirmSettleReopenCancel(t *testing.T) {
	repo := newFakeEntryRepo()
	ev := &fakeEventRepo{}
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{}, ev)

	// Confirm grava confirmed com paid_at e valor.
	e := seedEntry(repo, dom.StatusPrevista)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{WorkspaceID: e.WorkspaceID, ID: e.ID}); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got := ev.last(t)
	if got.Event != dom.EventConfirmed || got.PaidAt == nil || got.PaidAmountCents == nil {
		t.Fatalf("confirmed incompleto: %+v", got)
	}
	if got.EntryID != e.ID {
		t.Fatalf("evento apontando para o lançamento errado")
	}

	// Reopen grava reopened.
	if _, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if got = ev.last(t); got.Event != dom.EventReopened {
		t.Fatalf("quer reopened, veio %s", got.Event)
	}

	// Settle grava settled.
	if _, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "pix",
	}); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if got = ev.last(t); got.Event != dom.EventSettled || got.PaidAt == nil {
		t.Fatalf("settled incompleto: %+v", got)
	}

	// Cancel grava cancelled com o motivo.
	e2 := seedEntry(repo, dom.StatusPrevista)
	e2.WorkspaceID = e.WorkspaceID
	repo.entries[e2.ID] = e2
	reason := "cobranca_indevida"
	if _, err := svc.Cancel(context.Background(), e2.WorkspaceID, e2.ID, &reason); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got = ev.last(t); got.Event != dom.EventCancelled || got.CancelReason == nil || *got.CancelReason != reason {
		t.Fatalf("cancelled incompleto: %+v", got)
	}

	// A trilha do primeiro lançamento tem os 3 eventos, em ordem.
	trail, err := svc.ListEvents(context.Background(), e.WorkspaceID, e.ID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(trail) != 3 {
		t.Fatalf("trilha = %d eventos, quer 3", len(trail))
	}
}

func TestEventDueDateChangedPreservesOriginal(t *testing.T) {
	repo := newFakeEntryRepo()
	ev := &fakeEventRepo{}
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{}, ev)
	e := seedEntry(repo, dom.StatusPrevista)

	newDue := e.DueDate.AddDate(0, 0, 15)
	if _, _, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: string(e.Kind),
		AmountCents: e.AmountCents, DueDate: newDue, Description: e.Description,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := ev.last(t)
	if got.Event != dom.EventDueDateChanged {
		t.Fatalf("quer due_date_changed, veio %s", got.Event)
	}
	if got.OldDueDate == nil || !got.OldDueDate.Equal(e.DueDate) {
		t.Fatalf("vencimento original não preservado: %v", got.OldDueDate)
	}
	if got.NewDueDate == nil || !got.NewDueDate.Equal(newDue) {
		t.Fatalf("vencimento novo errado: %v", got.NewDueDate)
	}
}

// --- Imutabilidade de lançamento realizado (fase 4) ---

func TestUpdateRealizedRejectsFactChanges(t *testing.T) {
	base := func() (*fakeEntryRepo, *dom.FinancialEntry, *FinancialEntryService) {
		repo := newFakeEntryRepo()
		e := seedEntry(repo, dom.StatusRealizada)
		return repo, e, NewFinancialEntryService(repo, fakeCategoryRepo{})
	}
	valid := func(e *dom.FinancialEntry) UpdateEntryInput {
		return UpdateEntryInput{
			WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: string(e.Kind),
			AmountCents: e.AmountCents, DueDate: e.DueDate, Description: e.Description,
		}
	}

	// Valor.
	_, e, svc := base()
	in := valid(e)
	in.AmountCents = e.AmountCents + 1
	if _, _, err := svc.Update(context.Background(), in); err == nil {
		t.Error("mudar valor de lançamento pago deveria ser rejeitado")
	}
	// Vencimento.
	_, e, svc = base()
	in = valid(e)
	in.DueDate = e.DueDate.AddDate(0, 1, 0)
	if _, _, err := svc.Update(context.Background(), in); err == nil {
		t.Error("mudar vencimento de lançamento pago deveria ser rejeitado")
	}
	// Natureza.
	_, e, svc = base()
	in = valid(e)
	in.Kind = "credit"
	if _, _, err := svc.Update(context.Background(), in); err == nil {
		t.Error("mudar natureza de lançamento pago deveria ser rejeitado")
	}
	// Status por edição (deve usar Reopen/Cancel).
	_, e, svc = base()
	in = valid(e)
	in.Status = "prevista"
	if _, _, err := svc.Update(context.Background(), in); err == nil {
		t.Error("mudar status via edição deveria ser rejeitado")
	}
}

func TestUpdateRealizedAllowsLabels(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusRealizada)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	notes := "pago no débito automático"
	got, _, err := svc.Update(context.Background(), UpdateEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: string(e.Kind),
		AmountCents: e.AmountCents, DueDate: e.DueDate,
		Description: "Conta de luz — apto Brusque",
		Notes:       &notes,
	})
	if err != nil {
		t.Fatalf("editar rótulos de lançamento pago deve ser permitido: %v", err)
	}
	if got.Description != "Conta de luz — apto Brusque" || got.Notes == nil {
		t.Fatalf("rótulos não aplicados: %+v", got)
	}
}

// Rotina automática (sem ator no context): evento sai com actor nulo.
func TestEventWithoutActorIsNil(t *testing.T) {
	repo := newFakeEntryRepo()
	ev := &fakeEventRepo{}
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{}, ev)
	e := seedEntry(repo, dom.StatusPrevista)

	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{WorkspaceID: e.WorkspaceID, ID: e.ID}); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if got := ev.last(t); got.ActorUserID != nil {
		t.Fatalf("sem ator no context, actor deve ser nulo: %v", got.ActorUserID)
	}
	if ev.last(t).CreatedAt.After(time.Now().UTC().Add(time.Minute)) {
		t.Fatal("created_at implausível")
	}
}
