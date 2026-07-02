package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// fakeEntryRepo implementa dom.FinancialEntryRepository em memória para os testes.
type fakeEntryRepo struct {
	entries      map[uuid.UUID]*dom.FinancialEntry
	cascadeCalls []struct {
		ParentID uuid.UUID
		Status   dom.Status
		PaidAt   *time.Time
	}
}

func newFakeEntryRepo() *fakeEntryRepo {
	return &fakeEntryRepo{entries: map[uuid.UUID]*dom.FinancialEntry{}}
}

func (f *fakeEntryRepo) Create(_ context.Context, e *dom.FinancialEntry) error {
	f.entries[e.ID] = e
	return nil
}

func (f *fakeEntryRepo) CreateBatch(_ context.Context, es []*dom.FinancialEntry) error {
	for _, e := range es {
		f.entries[e.ID] = e
	}
	return nil
}

func (f *fakeEntryRepo) GetByID(_ context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	e, ok := f.entries[id]
	if !ok || e.WorkspaceID != workspaceID {
		return nil, dom.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (f *fakeEntryRepo) Update(_ context.Context, e *dom.FinancialEntry) error {
	if _, ok := f.entries[e.ID]; !ok {
		return dom.ErrNotFound
	}
	cp := *e
	f.entries[e.ID] = &cp
	return nil
}

func (f *fakeEntryRepo) SoftDelete(_ context.Context, _, id uuid.UUID) error {
	delete(f.entries, id)
	return nil
}

func (f *fakeEntryRepo) List(_ context.Context, _ uuid.UUID, _ dom.FinancialEntryFilter, _, _ int) ([]dom.FinancialEntry, int64, error) {
	return nil, 0, nil
}

func (f *fakeEntryRepo) ListRecurrenceFrontiers(_ context.Context) ([]dom.FinancialEntry, error) {
	return nil, nil
}

func (f *fakeEntryRepo) CascadeStatusToChildren(_ context.Context, _, parentID uuid.UUID, status dom.Status, paidAt *time.Time) error {
	f.cascadeCalls = append(f.cascadeCalls, struct {
		ParentID uuid.UUID
		Status   dom.Status
		PaidAt   *time.Time
	}{parentID, status, paidAt})
	return nil
}

// fakeCategoryRepo aceita qualquer slug (validação dinâmica coberta em testes próprios).
type fakeCategoryRepo struct{}

func (fakeCategoryRepo) Create(context.Context, *dom.ExpenseCategory) error        { return nil }
func (fakeCategoryRepo) CreateBatch(context.Context, []*dom.ExpenseCategory) error { return nil }
func (fakeCategoryRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*dom.ExpenseCategory, error) {
	return nil, dom.ErrNotFound
}
func (fakeCategoryRepo) ExistsBySlug(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}
func (fakeCategoryRepo) Update(context.Context, *dom.ExpenseCategory) error { return nil }
func (fakeCategoryRepo) SoftDelete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (fakeCategoryRepo) List(context.Context, uuid.UUID) ([]dom.ExpenseCategory, error) {
	return nil, nil
}

func seedEntry(repo *fakeEntryRepo, status dom.Status) *dom.FinancialEntry {
	e := &dom.FinancialEntry{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Kind:        dom.KindDebit,
		Status:      status,
		AmountCents: 10_000,
		DueDate:     time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Description: "conta de luz",
	}
	repo.entries[e.ID] = e
	return e
}

func TestSettleDefaultsAndCascade(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	got, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "pix",
	})
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if got.Status != dom.StatusRealizada {
		t.Fatalf("status = %s, quer realizada", got.Status)
	}
	if got.PaidAmountCents == nil || *got.PaidAmountCents != e.AmountCents {
		t.Fatalf("paid_amount default deve ser amount_cents (%d), veio %v", e.AmountCents, got.PaidAmountCents)
	}
	if got.PaidAt == nil {
		t.Fatal("paid_at default deve ser preenchido")
	}
	if got.PaymentMethod == nil || *got.PaymentMethod != dom.PaymentPix {
		t.Fatalf("payment_method = %v, quer pix", got.PaymentMethod)
	}
	if len(repo.cascadeCalls) != 1 || repo.cascadeCalls[0].ParentID != e.ID || repo.cascadeCalls[0].Status != dom.StatusRealizada {
		t.Fatalf("cascata esperada para o pai %s como realizada, veio %+v", e.ID, repo.cascadeCalls)
	}
}

func TestSettlePaidAmountCanDiffer(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paid := int64(10_550) // juros
	got, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "boleto", PaidAmountCents: &paid,
	})
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if *got.PaidAmountCents != paid {
		t.Fatalf("paid_amount = %d, quer %d", *got.PaidAmountCents, paid)
	}
	if got.AmountCents != 10_000 {
		t.Fatalf("amount original deve ser preservado, veio %d", got.AmountCents)
	}
}

func TestSettleRejectsCancelada(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusCancelada)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	_, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "pix",
	})
	if !errors.Is(err, dom.ErrValidation) {
		t.Fatalf("quer erro de validação para cancelada, veio %v", err)
	}
}

func TestSettleRejectsInvalidMethod(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	_, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "cheque",
	})
	if !errors.Is(err, dom.ErrValidation) {
		t.Fatalf("quer erro de validação para método desconhecido, veio %v", err)
	}
}

func TestSettleCartaoCreditoExigeCartao(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	_, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "cartao_credito",
	})
	if !errors.Is(err, dom.ErrValidation) {
		t.Fatalf("quer erro de validação sem card_id, veio %v", err)
	}

	cardID := uuid.New()
	if _, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "cartao_credito", CardID: &cardID,
	}); err != nil {
		t.Fatalf("com card_id deve liquidar, veio %v", err)
	}
}

func TestConfirmAndCancelCascade(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Confirm(context.Background(), e.WorkspaceID, e.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := svc.Cancel(context.Background(), e.WorkspaceID, e.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if len(repo.cascadeCalls) != 2 {
		t.Fatalf("Confirm e Cancel devem cascatear (2 chamadas), veio %d", len(repo.cascadeCalls))
	}
	if repo.cascadeCalls[0].Status != dom.StatusRealizada || repo.cascadeCalls[1].Status != dom.StatusCancelada {
		t.Fatalf("cascatas com status errado: %+v", repo.cascadeCalls)
	}
}
