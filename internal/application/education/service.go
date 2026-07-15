// Package education orquestra as regras de negócio de Educação / Material Escolar.
package education

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/education"
)

// Service orquestra o módulo de educação.
type Service struct {
	repo dom.Repository
}

func NewService(repo dom.Repository) *Service {
	return &Service{repo: repo}
}

// ─── Enrollments ─────────────────────────────────────────────────────────────

type CreateEnrollmentInput struct {
	WorkspaceID        uuid.UUID
	MemberID           uuid.UUID
	SchoolYear         int
	Stage              string
	SchoolName         *string
	Grade              *string
	Shift              *string
	MonthlyFeeCents    int64
	EnrollmentFeeCents int64
	Notes              *string
}

type UpdateEnrollmentInput struct {
	WorkspaceID        uuid.UUID
	ID                 uuid.UUID
	MemberID           uuid.UUID
	SchoolYear         int
	Stage              string
	SchoolName         *string
	Grade              *string
	Shift              *string
	MonthlyFeeCents    int64
	EnrollmentFeeCents int64
	Notes              *string
}

func (s *Service) CreateEnrollment(ctx context.Context, in CreateEnrollmentInput) (*dom.SchoolEnrollment, error) {
	now := time.Now().UTC()
	e := &dom.SchoolEnrollment{
		ID:                 uuid.New(),
		WorkspaceID:        in.WorkspaceID,
		MemberID:           in.MemberID,
		SchoolYear:         in.SchoolYear,
		Stage:              dom.Stage(in.Stage),
		SchoolName:         in.SchoolName,
		Grade:              in.Grade,
		Shift:              toShift(in.Shift),
		MonthlyFeeCents:    in.MonthlyFeeCents,
		EnrollmentFeeCents: in.EnrollmentFeeCents,
		Notes:              in.Notes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateEnrollment(ctx, e); err != nil {
		return nil, err
	}
	s.enrichEnrollments(ctx, in.WorkspaceID, []*dom.SchoolEnrollment{e})
	return e, nil
}

func (s *Service) GetEnrollment(ctx context.Context, workspaceID, id uuid.UUID) (*dom.SchoolEnrollment, error) {
	e, err := s.repo.GetEnrollment(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	s.enrichEnrollments(ctx, workspaceID, []*dom.SchoolEnrollment{e})
	return e, nil
}

func (s *Service) ListEnrollments(ctx context.Context, workspaceID uuid.UUID, memberID *uuid.UUID, schoolYear *int) ([]dom.SchoolEnrollment, error) {
	items, err := s.repo.ListEnrollments(ctx, workspaceID, dom.ListEnrollmentsParams{MemberID: memberID, SchoolYear: schoolYear})
	if err != nil {
		return nil, err
	}
	ptrs := make([]*dom.SchoolEnrollment, len(items))
	for i := range items {
		ptrs[i] = &items[i]
	}
	s.enrichEnrollments(ctx, workspaceID, ptrs)
	return items, nil
}

func (s *Service) UpdateEnrollment(ctx context.Context, in UpdateEnrollmentInput) (*dom.SchoolEnrollment, error) {
	e, err := s.repo.GetEnrollment(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	e.MemberID = in.MemberID
	e.SchoolYear = in.SchoolYear
	e.Stage = dom.Stage(in.Stage)
	e.SchoolName = in.SchoolName
	e.Grade = in.Grade
	e.Shift = toShift(in.Shift)
	e.MonthlyFeeCents = in.MonthlyFeeCents
	e.EnrollmentFeeCents = in.EnrollmentFeeCents
	e.Notes = in.Notes
	e.UpdatedAt = time.Now().UTC()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateEnrollment(ctx, e); err != nil {
		return nil, err
	}
	s.enrichEnrollments(ctx, in.WorkspaceID, []*dom.SchoolEnrollment{e})
	return e, nil
}

func (s *Service) DeleteEnrollment(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteEnrollment(ctx, workspaceID, id)
}

// ─── Supply lists ────────────────────────────────────────────────────────────

type CreateListInput struct {
	WorkspaceID  uuid.UUID
	EnrollmentID uuid.UUID
	Title        string
	Status       string
	Notes        *string
}

type UpdateListInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	Title       string
	Status      string
	Notes       *string
}

func (s *Service) CreateList(ctx context.Context, in CreateListInput) (*dom.SchoolSupplyList, error) {
	// Garante que a matrícula existe e pertence ao workspace.
	if _, err := s.repo.GetEnrollment(ctx, in.WorkspaceID, in.EnrollmentID); err != nil {
		return nil, err
	}
	status := dom.ListStatus(in.Status)
	if status == "" {
		status = dom.ListStatusPlanejada
	}
	now := time.Now().UTC()
	l := &dom.SchoolSupplyList{
		ID:           uuid.New(),
		WorkspaceID:  in.WorkspaceID,
		EnrollmentID: in.EnrollmentID,
		Title:        in.Title,
		Status:       status,
		Notes:        in.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateList(ctx, l); err != nil {
		return nil, err
	}
	s.enrichLists(ctx, in.WorkspaceID, []*dom.SchoolSupplyList{l})
	return l, nil
}

func (s *Service) GetList(ctx context.Context, workspaceID, id uuid.UUID) (*dom.SchoolSupplyList, error) {
	l, err := s.repo.GetList(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	s.enrichLists(ctx, workspaceID, []*dom.SchoolSupplyList{l})
	return l, nil
}

func (s *Service) ListSupplyLists(ctx context.Context, workspaceID uuid.UUID, enrollmentID *uuid.UUID, schoolYear *int, status string) ([]dom.SchoolSupplyList, error) {
	items, err := s.repo.ListSupplyLists(ctx, workspaceID, dom.ListSupplyListsParams{
		EnrollmentID: enrollmentID,
		SchoolYear:   schoolYear,
		Status:       status,
	})
	if err != nil {
		return nil, err
	}
	ptrs := make([]*dom.SchoolSupplyList, len(items))
	for i := range items {
		ptrs[i] = &items[i]
	}
	s.enrichLists(ctx, workspaceID, ptrs)
	return items, nil
}

func (s *Service) UpdateList(ctx context.Context, in UpdateListInput) (*dom.SchoolSupplyList, error) {
	l, err := s.repo.GetList(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	l.Title = in.Title
	if in.Status != "" {
		l.Status = dom.ListStatus(in.Status)
	}
	l.Notes = in.Notes
	l.UpdatedAt = time.Now().UTC()
	if err := l.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateList(ctx, l); err != nil {
		return nil, err
	}
	s.enrichLists(ctx, in.WorkspaceID, []*dom.SchoolSupplyList{l})
	return l, nil
}

func (s *Service) DeleteList(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteList(ctx, workspaceID, id)
}

// ─── Supply items ────────────────────────────────────────────────────────────

type AddItemInput struct {
	WorkspaceID         uuid.UUID
	ListID              uuid.UUID
	Name                string
	Category            string
	Quantity            float64
	ReferencePriceCents int64
	Purchased           bool
	PaidPriceCents      int64
	PurchasedAt         *time.Time
	Store               *string
	Notes               *string
}

type UpdateItemInput struct {
	WorkspaceID         uuid.UUID
	ListID              uuid.UUID
	ItemID              uuid.UUID
	Name                string
	Category            string
	Quantity            float64
	ReferencePriceCents int64
	Purchased           bool
	PaidPriceCents      int64
	PurchasedAt         *time.Time
	Store               *string
	Notes               *string
}

type PurchaseItemInput struct {
	WorkspaceID    uuid.UUID
	ListID         uuid.UUID
	ItemID         uuid.UUID
	PaidPriceCents int64
	PurchasedAt    *time.Time
	Store          *string
}

func (s *Service) AddItem(ctx context.Context, in AddItemInput) (*dom.SchoolSupplyItem, error) {
	// Garante que a lista existe e pertence ao workspace.
	if _, err := s.repo.GetList(ctx, in.WorkspaceID, in.ListID); err != nil {
		return nil, err
	}
	category := dom.ItemCategory(in.Category)
	if category == "" {
		category = dom.ItemCategoryOutros
	}
	qty := in.Quantity
	if qty == 0 {
		qty = 1
	}
	now := time.Now().UTC()
	item := &dom.SchoolSupplyItem{
		ID:                  uuid.New(),
		WorkspaceID:         in.WorkspaceID,
		ListID:              in.ListID,
		Name:                in.Name,
		Category:            category,
		Quantity:            qty,
		ReferencePriceCents: in.ReferencePriceCents,
		Purchased:           in.Purchased,
		PaidPriceCents:      in.PaidPriceCents,
		PurchasedAt:         in.PurchasedAt,
		Store:               in.Store,
		Notes:               in.Notes,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, in UpdateItemInput) (*dom.SchoolSupplyItem, error) {
	item, err := s.getItemInList(ctx, in.WorkspaceID, in.ListID, in.ItemID)
	if err != nil {
		return nil, err
	}
	item.Name = in.Name
	if in.Category != "" {
		item.Category = dom.ItemCategory(in.Category)
	}
	if in.Quantity > 0 {
		item.Quantity = in.Quantity
	}
	item.ReferencePriceCents = in.ReferencePriceCents
	item.Purchased = in.Purchased
	item.PaidPriceCents = in.PaidPriceCents
	item.PurchasedAt = in.PurchasedAt
	item.Store = in.Store
	item.Notes = in.Notes
	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// PurchaseItem marca um item como comprado, registrando valor pago, data e loja.
func (s *Service) PurchaseItem(ctx context.Context, in PurchaseItemInput) (*dom.SchoolSupplyItem, error) {
	item, err := s.getItemInList(ctx, in.WorkspaceID, in.ListID, in.ItemID)
	if err != nil {
		return nil, err
	}
	item.Purchased = true
	item.PaidPriceCents = in.PaidPriceCents
	if in.PurchasedAt != nil {
		item.PurchasedAt = in.PurchasedAt
	} else if item.PurchasedAt == nil {
		now := time.Now().UTC()
		item.PurchasedAt = &now
	}
	if in.Store != nil {
		item.Store = in.Store
	}
	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) DeleteItem(ctx context.Context, workspaceID, listID, itemID uuid.UUID) error {
	if _, err := s.getItemInList(ctx, workspaceID, listID, itemID); err != nil {
		return err
	}
	return s.repo.DeleteItem(ctx, workspaceID, itemID)
}

func (s *Service) getItemInList(ctx context.Context, workspaceID, listID, itemID uuid.UUID) (*dom.SchoolSupplyItem, error) {
	item, err := s.repo.GetItem(ctx, workspaceID, itemID)
	if err != nil {
		return nil, err
	}
	if item.ListID != listID {
		return nil, dom.ErrNotFound
	}
	return item, nil
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

// Dashboard consolida os indicadores de educação. Se schoolYear <= 0, usa o
// maior ano letivo com matrículas cadastradas.
func (s *Service) Dashboard(ctx context.Context, workspaceID uuid.UUID, schoolYear int) (*dom.Dashboard, error) {
	enrollments, err := s.repo.AllEnrollments(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	lists, err := s.repo.AllLists(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.AllItems(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.MemberNames(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if schoolYear <= 0 {
		for _, e := range enrollments {
			if e.SchoolYear > schoolYear {
				schoolYear = e.SchoolYear
			}
		}
	}

	// Mapas de apoio.
	enrollByID := make(map[string]dom.SchoolEnrollment, len(enrollments))
	for _, e := range enrollments {
		enrollByID[e.ID.String()] = e
	}
	listYear := make(map[string]int, len(lists))    // list_id → school_year
	listMember := make(map[string]string, len(lists)) // list_id → member_id
	for _, l := range lists {
		if e, ok := enrollByID[l.EnrollmentID.String()]; ok {
			listYear[l.ID.String()] = e.SchoolYear
			listMember[l.ID.String()] = e.MemberID.String()
		}
	}

	d := &dom.Dashboard{SchoolYear: schoolYear}

	// Mensalidades e matrícula do ano.
	for _, e := range enrollments {
		if e.SchoolYear == schoolYear {
			d.MonthlyFeesCents += e.MonthlyFeeCents
			d.EnrollmentFeesCents += e.EnrollmentFeeCents
		}
	}

	// Contagem de listas do ano.
	for _, l := range lists {
		if listYear[l.ID.String()] == schoolYear {
			d.ListCount++
		}
	}

	// Agregações sobre itens do ano.
	memberAgg := make(map[string]*dom.MemberSpend)
	catAgg := make(map[string]*dom.CategoryAvg)
	var refPurchasedCents int64

	for _, it := range items {
		lid := it.ListID.String()
		year, ok := listYear[lid]
		if !ok || year != schoolYear {
			continue
		}
		refCents := int64(float64(it.ReferencePriceCents) * it.Quantity)
		d.TotalReferenceCents += refCents
		d.ItemCount++

		memberID := listMember[lid]
		ms := memberAgg[memberID]
		if ms == nil {
			ms = &dom.MemberSpend{MemberID: memberID, MemberName: names[memberID]}
			memberAgg[memberID] = ms
		}
		ms.ItemCount++

		ca := catAgg[string(it.Category)]
		if ca == nil {
			ca = &dom.CategoryAvg{Category: string(it.Category)}
			catAgg[string(it.Category)] = ca
		}
		ca.ItemCount++

		if it.Purchased {
			d.PurchasedCount++
			d.TotalPaidCents += it.PaidPriceCents
			refPurchasedCents += refCents
			ms.PurchasedCount++
			ms.TotalPaidCents += it.PaidPriceCents
			ca.PurchasedCount++
			ca.TotalPaidCents += it.PaidPriceCents
		}
	}

	if d.ItemCount > 0 {
		d.PurchasedPct = round1(float64(d.PurchasedCount) / float64(d.ItemCount) * 100)
	}
	// Economia/estouro sobre os itens efetivamente comprados.
	d.SavingsCents = refPurchasedCents - d.TotalPaidCents
	if refPurchasedCents > 0 {
		d.SavingsPct = round1(float64(d.SavingsCents) / float64(refPurchasedCents) * 100)
	}

	// Por membro.
	for _, ms := range memberAgg {
		if ms.ItemCount > 0 {
			ms.PurchasedPct = round1(float64(ms.PurchasedCount) / float64(ms.ItemCount) * 100)
		}
		d.ByMember = append(d.ByMember, *ms)
	}
	sort.Slice(d.ByMember, func(i, j int) bool {
		return d.ByMember[i].TotalPaidCents > d.ByMember[j].TotalPaidCents
	})

	// Por categoria (custo médio por item comprado).
	for _, ca := range catAgg {
		if ca.PurchasedCount > 0 {
			ca.AvgPaidCents = ca.TotalPaidCents / int64(ca.PurchasedCount)
		}
		d.ByCategory = append(d.ByCategory, *ca)
	}
	sort.Slice(d.ByCategory, func(i, j int) bool {
		return d.ByCategory[i].TotalPaidCents > d.ByCategory[j].TotalPaidCents
	})

	// Evolução anual (mensalidades anualizadas + matrícula + material pago).
	yearAgg := make(map[int]*dom.YearSpend)
	for _, e := range enrollments {
		ys := yearAgg[e.SchoolYear]
		if ys == nil {
			ys = &dom.YearSpend{SchoolYear: e.SchoolYear}
			yearAgg[e.SchoolYear] = ys
		}
		ys.MonthlyFeesCents += e.MonthlyFeeCents * 12
		ys.EnrollmentFeesCents += e.EnrollmentFeeCents
	}
	for _, it := range items {
		if !it.Purchased {
			continue
		}
		year, ok := listYear[it.ListID.String()]
		if !ok {
			continue
		}
		ys := yearAgg[year]
		if ys == nil {
			ys = &dom.YearSpend{SchoolYear: year}
			yearAgg[year] = ys
		}
		ys.SuppliesPaidCents += it.PaidPriceCents
	}
	for _, ys := range yearAgg {
		ys.TotalCents = ys.MonthlyFeesCents + ys.EnrollmentFeesCents + ys.SuppliesPaidCents
		d.AnnualEvolution = append(d.AnnualEvolution, *ys)
	}
	sort.Slice(d.AnnualEvolution, func(i, j int) bool {
		return d.AnnualEvolution[i].SchoolYear < d.AnnualEvolution[j].SchoolYear
	})

	return d, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Service) enrichEnrollments(ctx context.Context, workspaceID uuid.UUID, items []*dom.SchoolEnrollment) {
	if len(items) == 0 {
		return
	}
	names, err := s.repo.MemberNames(ctx, workspaceID)
	if err != nil {
		return
	}
	for _, e := range items {
		if n, ok := names[e.MemberID.String()]; ok {
			nn := n
			e.MemberName = &nn
		}
	}
}

func (s *Service) enrichLists(ctx context.Context, workspaceID uuid.UUID, items []*dom.SchoolSupplyList) {
	if len(items) == 0 {
		return
	}
	enrollments, err := s.repo.AllEnrollments(ctx, workspaceID)
	if err != nil {
		return
	}
	names, err := s.repo.MemberNames(ctx, workspaceID)
	if err != nil {
		return
	}
	enrollByID := make(map[string]dom.SchoolEnrollment, len(enrollments))
	for _, e := range enrollments {
		enrollByID[e.ID.String()] = e
	}
	for _, l := range items {
		if e, ok := enrollByID[l.EnrollmentID.String()]; ok {
			mid := e.MemberID
			year := e.SchoolYear
			l.MemberID = &mid
			l.SchoolYear = &year
			if n, ok := names[e.MemberID.String()]; ok {
				nn := n
				l.MemberName = &nn
			}
		}
	}
}

func toShift(s *string) *dom.Shift {
	if s == nil || *s == "" {
		return nil
	}
	sh := dom.Shift(*s)
	return &sh
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}
