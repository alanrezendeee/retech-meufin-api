package finance

import (
	"context"
	"errors"
	"sort"
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

func (f *fakeEntryRepo) ListInvoiceInstallments(_ context.Context, workspaceID uuid.UUID) ([]dom.FinancialEntry, error) {
	var out []dom.FinancialEntry
	for _, e := range f.entries {
		if e.WorkspaceID == workspaceID && e.ParentID != nil && e.InstallmentNumber != nil && e.InstallmentTotal != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (f *fakeEntryRepo) ListFutureGroupSiblings(_ context.Context, workspaceID, groupID uuid.UUID, after time.Time, excludeID uuid.UUID) ([]dom.FinancialEntry, error) {
	out := []dom.FinancialEntry{}
	for _, e := range f.entries {
		if e.WorkspaceID == workspaceID && e.RecurrenceGroupID != nil && *e.RecurrenceGroupID == groupID &&
			e.Status == dom.StatusPrevista && e.DueDate.After(after) && e.ID != excludeID {
			out = append(out, *e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueDate.Before(out[j].DueDate) })
	return out, nil
}

func (f *fakeEntryRepo) ListResiduals(_ context.Context, workspaceID, originID uuid.UUID) ([]dom.FinancialEntry, error) {
	var out []dom.FinancialEntry
	for _, e := range f.entries {
		if e.WorkspaceID == workspaceID && e.ResidualOfID != nil && *e.ResidualOfID == originID {
			out = append(out, *e)
		}
	}
	return out, nil
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

func TestConfirmWithDiscount(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	discount := int64(1_500)
	reason := "pagamento_antecipado"
	got, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID,
		DiscountCents: &discount, DiscountReason: &reason,
	})
	if err != nil {
		t.Fatalf("Confirm com desconto: %v", err)
	}
	if got.PaidAmountCents == nil || *got.PaidAmountCents != e.AmountCents-discount {
		t.Fatalf("paid_amount deve ser amount - desconto (%d), veio %v", e.AmountCents-discount, got.PaidAmountCents)
	}
	if got.DiscountCents == nil || *got.DiscountCents != discount {
		t.Fatalf("discount_cents não persistido: %v", got.DiscountCents)
	}
	if got.DiscountReason == nil || *got.DiscountReason != reason {
		t.Fatalf("discount_reason não persistido: %v", got.DiscountReason)
	}
	if got.PaidAt == nil {
		t.Fatal("confirm com desconto deve registrar paid_at")
	}
}

func TestConfirmDiscountValidation(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	cases := []struct {
		name     string
		discount int64
		reason   string
	}{
		{"desconto sem motivo", 1_000, ""},
		{"motivo fora do catálogo", 1_000, "achei_bonito"},
		{"desconto >= valor", 10_000, "pagamento_antecipado"},
		{"desconto negativo", -100, "pagamento_antecipado"},
	}
	for _, tc := range cases {
		e := seedEntry(repo, dom.StatusPrevista)
		var reason *string
		if tc.reason != "" {
			reason = &tc.reason
		}
		_, err := svc.Confirm(context.Background(), ConfirmEntryInput{
			WorkspaceID: e.WorkspaceID, ID: e.ID,
			DiscountCents: &tc.discount, DiscountReason: reason,
		})
		var vErr *dom.ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("%s: quer ValidationError, veio %v", tc.name, err)
		}
	}
}

func TestSettleWithDiscountDefaultsPaidAmount(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	discount := int64(2_000)
	reason := "negociacao"
	got, err := svc.Settle(context.Background(), SettleEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaymentMethod: "pix",
		DiscountCents: &discount, DiscountReason: &reason,
	})
	if err != nil {
		t.Fatalf("Settle com desconto: %v", err)
	}
	if got.PaidAmountCents == nil || *got.PaidAmountCents != e.AmountCents-discount {
		t.Fatalf("paid_amount deve ser amount - desconto (%d), veio %v", e.AmountCents-discount, got.PaidAmountCents)
	}
	if got.DiscountReason == nil || *got.DiscountReason != reason {
		t.Fatalf("discount_reason não persistido: %v", got.DiscountReason)
	}
}

func TestReopenClearsDiscount(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	discount := int64(1_500)
	reason := "cupom"
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID,
		DiscountCents: &discount, DiscountReason: &reason,
	}); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if got.DiscountCents != nil || got.DiscountReason != nil {
		t.Fatalf("reopen deve limpar desconto, veio %v/%v", got.DiscountCents, got.DiscountReason)
	}
	if got.PaidAt != nil || got.PaidAmountCents != nil {
		t.Fatalf("reopen deve limpar liquidação, veio %v/%v", got.PaidAt, got.PaidAmountCents)
	}
}

func findResidual(repo *fakeEntryRepo, originID uuid.UUID) *dom.FinancialEntry {
	for _, e := range repo.entries {
		if e.ResidualOfID != nil && *e.ResidualOfID == originID {
			return e
		}
	}
	return nil
}

func TestConfirmPartialCreatesResidual(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista) // 10_000
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paid := int64(1_500)
	got, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAmountCents: &paid,
	})
	if err != nil {
		t.Fatalf("Confirm parcial: %v", err)
	}
	if got.Status != dom.StatusRealizada || got.PaidAmountCents == nil || *got.PaidAmountCents != paid {
		t.Fatalf("origem deve ficar realizada com paid=%d, veio %+v", paid, got)
	}
	res := findResidual(repo, e.ID)
	if res == nil {
		t.Fatal("residual não criado")
	}
	if res.AmountCents != 8_500 {
		t.Fatalf("residual deve ser 8500, veio %d", res.AmountCents)
	}
	if res.Status != dom.StatusPrevista {
		t.Fatalf("residual deve nascer prevista, veio %s", res.Status)
	}
	if !res.DueDate.Equal(e.DueDate) {
		t.Fatalf("residual deve vencer na data original %v, veio %v", e.DueDate, res.DueDate)
	}
}

func TestConfirmPartialWithDiscount(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista) // 10_000
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paid := int64(7_000)
	discount := int64(1_000)
	reason := "negociacao"
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID,
		PaidAmountCents: &paid, DiscountCents: &discount, DiscountReason: &reason,
	}); err != nil {
		t.Fatalf("Confirm parcial+desconto: %v", err)
	}
	res := findResidual(repo, e.ID)
	if res == nil || res.AmountCents != 2_000 {
		t.Fatalf("residual deve ser amount - desconto - pago = 2000, veio %+v", res)
	}
}

func TestConfirmPartialValidation(t *testing.T) {
	repo := newFakeEntryRepo()
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	// pago > devido não suportado (juros/multa é fase futura)
	e := seedEntry(repo, dom.StatusPrevista)
	over := int64(12_000)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAmountCents: &over,
	}); err == nil {
		t.Fatal("pago > devido deveria falhar")
	}

	// pago igual ao devido não gera residual
	e2 := seedEntry(repo, dom.StatusPrevista)
	full := int64(10_000)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e2.WorkspaceID, ID: e2.ID, PaidAmountCents: &full,
	}); err != nil {
		t.Fatalf("pago integral deve liquidar: %v", err)
	}
	if findResidual(repo, e2.ID) != nil {
		t.Fatal("pago integral não deve gerar residual")
	}

	// fatura de cartão bloqueia parcial
	cartao := "cartao"
	inv := &dom.FinancialEntry{
		ID: uuid.New(), WorkspaceID: uuid.New(), Kind: dom.KindDebit,
		Status: dom.StatusPrevista, AmountCents: 10_000, Type: &cartao,
		DueDate: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), Description: "fatura",
	}
	repo.entries[inv.ID] = inv
	part := int64(4_000)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: inv.WorkspaceID, ID: inv.ID, PaidAmountCents: &part,
	}); err == nil {
		t.Fatal("parcial em fatura deveria falhar")
	}
}

func TestReopenPartialRemovesResidual(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paid := int64(1_500)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAmountCents: &paid,
	}); err != nil {
		t.Fatalf("Confirm parcial: %v", err)
	}
	if _, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if findResidual(repo, e.ID) != nil {
		t.Fatal("reopen deve remover o residual previsto")
	}
}

func TestReopenBlockedByPaidResidual(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paid := int64(1_500)
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAmountCents: &paid,
	}); err != nil {
		t.Fatalf("Confirm parcial: %v", err)
	}
	res := findResidual(repo, e.ID)
	if res == nil {
		t.Fatal("residual não criado")
	}
	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: res.WorkspaceID, ID: res.ID,
	}); err != nil {
		t.Fatalf("Confirm do residual: %v", err)
	}
	if _, err := svc.Reopen(context.Background(), e.WorkspaceID, e.ID); err == nil {
		t.Fatal("reopen com residual pago deveria falhar")
	}
}

func TestConfirmWithPaidAt(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	paidAt := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	got, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, PaidAt: &paidAt,
	})
	if err != nil {
		t.Fatalf("Confirm com paid_at: %v", err)
	}
	if got.PaidAt == nil || !got.PaidAt.Equal(paidAt) {
		t.Fatalf("paid_at deve ser a data informada (%v), veio %v", paidAt, got.PaidAt)
	}
	if got.Status != dom.StatusRealizada {
		t.Fatalf("status deve ser realizada, veio %s", got.Status)
	}
	// paid_at informado junto com desconto
	e2 := seedEntry(repo, dom.StatusPrevista)
	discount := int64(500)
	reason := "pontualidade"
	got2, err := svc.Confirm(context.Background(), ConfirmEntryInput{
		WorkspaceID: e2.WorkspaceID, ID: e2.ID, PaidAt: &paidAt,
		DiscountCents: &discount, DiscountReason: &reason,
	})
	if err != nil {
		t.Fatalf("Confirm paid_at+desconto: %v", err)
	}
	if got2.PaidAt == nil || !got2.PaidAt.Equal(paidAt) {
		t.Fatalf("paid_at com desconto deve ser a data informada, veio %v", got2.PaidAt)
	}
}

func TestUpdateInstallmentsPreserveAndClear(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	three, ten := 3, 10
	e.InstallmentNumber = &three
	e.InstallmentTotal = &ten
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	base := UpdateEntryInput{
		WorkspaceID: e.WorkspaceID, ID: e.ID, Kind: string(dom.KindDebit),
		AmountCents: e.AmountCents, DueDate: e.DueDate, Description: e.Description,
	}

	// ausente preserva (edição genérica não envia)
	got, _, err := svc.Update(context.Background(), base)
	if err != nil {
		t.Fatalf("Update sem parcela: %v", err)
	}
	if got.InstallmentNumber == nil || *got.InstallmentNumber != 3 || got.InstallmentTotal == nil || *got.InstallmentTotal != 10 {
		t.Fatalf("parcela deve ser preservada, veio %v/%v", got.InstallmentNumber, got.InstallmentTotal)
	}

	// alterar número
	five := 5
	in := base
	in.InstallmentNumber = &five
	got, _, err = svc.Update(context.Background(), in)
	if err != nil {
		t.Fatalf("Update parcela 5: %v", err)
	}
	if *got.InstallmentNumber != 5 || *got.InstallmentTotal != 10 {
		t.Fatalf("quer 5/10, veio %v/%v", got.InstallmentNumber, got.InstallmentTotal)
	}

	// número > total rejeitado
	twenty := 20
	in = base
	in.InstallmentNumber = &twenty
	if _, _, err := svc.Update(context.Background(), in); err == nil {
		t.Fatal("parcela 20/10 deveria falhar")
	}

	// zero limpa
	zero := 0
	in = base
	in.InstallmentNumber = &zero
	in.InstallmentTotal = &zero
	got, _, err = svc.Update(context.Background(), in)
	if err != nil {
		t.Fatalf("Update limpando parcela: %v", err)
	}
	if got.InstallmentNumber != nil || got.InstallmentTotal != nil {
		t.Fatalf("zero deve limpar a parcela, veio %v/%v", got.InstallmentNumber, got.InstallmentTotal)
	}
}

func TestConfirmAndCancelCascade(t *testing.T) {
	repo := newFakeEntryRepo()
	e := seedEntry(repo, dom.StatusPrevista)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.Confirm(context.Background(), ConfirmEntryInput{WorkspaceID: e.WorkspaceID, ID: e.ID}); err != nil {
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
