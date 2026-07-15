package education

import (
	"context"
	"testing"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/education"
)

// fakeRepo é uma implementação em memória de dom.Repository para testes.
type fakeRepo struct {
	enrollments []dom.SchoolEnrollment
	lists       []dom.SchoolSupplyList
	items       []dom.SchoolSupplyItem
	names       map[string]string
}

func (f *fakeRepo) CreateEnrollment(ctx context.Context, e *dom.SchoolEnrollment) error { return nil }
func (f *fakeRepo) GetEnrollment(ctx context.Context, ws, id uuid.UUID) (*dom.SchoolEnrollment, error) {
	for i := range f.enrollments {
		if f.enrollments[i].ID == id {
			return &f.enrollments[i], nil
		}
	}
	return nil, dom.ErrNotFound
}
func (f *fakeRepo) ListEnrollments(ctx context.Context, ws uuid.UUID, p dom.ListEnrollmentsParams) ([]dom.SchoolEnrollment, error) {
	return f.enrollments, nil
}
func (f *fakeRepo) UpdateEnrollment(ctx context.Context, e *dom.SchoolEnrollment) error { return nil }
func (f *fakeRepo) DeleteEnrollment(ctx context.Context, ws, id uuid.UUID) error        { return nil }
func (f *fakeRepo) CreateList(ctx context.Context, l *dom.SchoolSupplyList) error        { return nil }
func (f *fakeRepo) GetList(ctx context.Context, ws, id uuid.UUID) (*dom.SchoolSupplyList, error) {
	return nil, dom.ErrNotFound
}
func (f *fakeRepo) ListSupplyLists(ctx context.Context, ws uuid.UUID, p dom.ListSupplyListsParams) ([]dom.SchoolSupplyList, error) {
	return f.lists, nil
}
func (f *fakeRepo) UpdateList(ctx context.Context, l *dom.SchoolSupplyList) error         { return nil }
func (f *fakeRepo) DeleteList(ctx context.Context, ws, id uuid.UUID) error                { return nil }
func (f *fakeRepo) CreateItem(ctx context.Context, i *dom.SchoolSupplyItem) error         { return nil }
func (f *fakeRepo) GetItem(ctx context.Context, ws, id uuid.UUID) (*dom.SchoolSupplyItem, error) {
	return nil, dom.ErrNotFound
}
func (f *fakeRepo) UpdateItem(ctx context.Context, i *dom.SchoolSupplyItem) error { return nil }
func (f *fakeRepo) DeleteItem(ctx context.Context, ws, id uuid.UUID) error        { return nil }
func (f *fakeRepo) AllEnrollments(ctx context.Context, ws uuid.UUID) ([]dom.SchoolEnrollment, error) {
	return f.enrollments, nil
}
func (f *fakeRepo) AllLists(ctx context.Context, ws uuid.UUID) ([]dom.SchoolSupplyList, error) {
	return f.lists, nil
}
func (f *fakeRepo) AllItems(ctx context.Context, ws uuid.UUID) ([]dom.SchoolSupplyItem, error) {
	return f.items, nil
}
func (f *fakeRepo) MemberNames(ctx context.Context, ws uuid.UUID) (map[string]string, error) {
	return f.names, nil
}

func TestDashboardAggregation(t *testing.T) {
	ws := uuid.New()
	member := uuid.New()
	enrollment := uuid.New()
	list := uuid.New()

	repo := &fakeRepo{
		names: map[string]string{member.String(): "João"},
		enrollments: []dom.SchoolEnrollment{{
			ID: enrollment, WorkspaceID: ws, MemberID: member, SchoolYear: 2026,
			Stage: dom.StageFundamental1, MonthlyFeeCents: 100000, EnrollmentFeeCents: 50000,
		}},
		lists: []dom.SchoolSupplyList{{
			ID: list, WorkspaceID: ws, EnrollmentID: enrollment, Title: "Lista 2026", Status: dom.ListStatusEmCompra,
		}},
		items: []dom.SchoolSupplyItem{
			// referência 2×1000 = 2000; comprado por 1800 → economia 200
			{ID: uuid.New(), WorkspaceID: ws, ListID: list, Name: "Caderno", Category: dom.ItemCategoryPapelaria,
				Quantity: 2, ReferencePriceCents: 1000, Purchased: true, PaidPriceCents: 1800},
			// referência 1×5000 = 5000; não comprado
			{ID: uuid.New(), WorkspaceID: ws, ListID: list, Name: "Mochila", Category: dom.ItemCategoryMochila,
				Quantity: 1, ReferencePriceCents: 5000, Purchased: false},
		},
	}

	svc := NewService(repo)
	d, err := svc.Dashboard(context.Background(), ws, 0) // 0 → maior ano (2026)
	if err != nil {
		t.Fatalf("dashboard erro: %v", err)
	}

	if d.SchoolYear != 2026 {
		t.Errorf("SchoolYear = %d, quero 2026", d.SchoolYear)
	}
	if d.TotalReferenceCents != 7000 {
		t.Errorf("TotalReferenceCents = %d, quero 7000", d.TotalReferenceCents)
	}
	if d.TotalPaidCents != 1800 {
		t.Errorf("TotalPaidCents = %d, quero 1800", d.TotalPaidCents)
	}
	if d.ItemCount != 2 || d.PurchasedCount != 1 {
		t.Errorf("ItemCount=%d PurchasedCount=%d, quero 2/1", d.ItemCount, d.PurchasedCount)
	}
	if d.PurchasedPct != 50 {
		t.Errorf("PurchasedPct = %v, quero 50", d.PurchasedPct)
	}
	// economia = referência dos comprados (2000) − pago (1800) = 200
	if d.SavingsCents != 200 {
		t.Errorf("SavingsCents = %d, quero 200", d.SavingsCents)
	}
	if d.MonthlyFeesCents != 100000 || d.EnrollmentFeesCents != 50000 {
		t.Errorf("fees mensal=%d matrícula=%d", d.MonthlyFeesCents, d.EnrollmentFeesCents)
	}
	if len(d.ByMember) != 1 || d.ByMember[0].MemberName != "João" || d.ByMember[0].TotalPaidCents != 1800 {
		t.Errorf("ByMember inesperado: %+v", d.ByMember)
	}
	// evolução anual: mensalidade anualizada 100000×12 + matrícula 50000 + material 1800
	if len(d.AnnualEvolution) != 1 {
		t.Fatalf("AnnualEvolution len = %d, quero 1", len(d.AnnualEvolution))
	}
	if got := d.AnnualEvolution[0].TotalCents; got != 1200000+50000+1800 {
		t.Errorf("AnnualEvolution total = %d, quero %d", got, 1200000+50000+1800)
	}
}
