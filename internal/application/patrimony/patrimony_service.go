// Package patrimony orquestra as regras de negócio de imóveis e impostos.
package patrimony

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/patrimony"
)

// Service orquestra imóveis, impostos e o overview do dashboard de patrimônio.
type Service struct {
	repo dom.Repository
}

func NewService(repo dom.Repository) *Service {
	return &Service{repo: repo}
}

// ─── Property CRUD ─────────────────────────────────────────────────────────────

type PropertyInput struct {
	WorkspaceID        uuid.UUID
	ID                 uuid.UUID // usado no Update
	Name               string
	PropertyType       string
	Address            *string
	City               *string
	State              *string
	ZipCode            *string
	RegistrationNumber *string
	AreaM2             *float64
	PurchaseDate       *time.Time
	PurchaseValueCents *int64
	CurrentValueCents  *int64
	Notes              *string
	Active             *bool
}

func (s *Service) CreateProperty(ctx context.Context, in PropertyInput) (*dom.Property, error) {
	now := time.Now().UTC()
	pt := dom.PropertyType(in.PropertyType)
	if pt == "" {
		pt = dom.PropertyCasa
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	p := &dom.Property{
		ID:                 uuid.New(),
		WorkspaceID:        in.WorkspaceID,
		Name:               in.Name,
		PropertyType:       pt,
		Address:            in.Address,
		City:               in.City,
		State:              in.State,
		ZipCode:            in.ZipCode,
		RegistrationNumber: in.RegistrationNumber,
		AreaM2:             in.AreaM2,
		PurchaseDate:       in.PurchaseDate,
		PurchaseValueCents: in.PurchaseValueCents,
		CurrentValueCents:  in.CurrentValueCents,
		Notes:              in.Notes,
		Active:             active,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateProperty(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetProperty(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Property, error) {
	return s.repo.GetProperty(ctx, workspaceID, id)
}

type ListPropertiesResult struct {
	Items []dom.Property
	Total int64
}

func (s *Service) ListProperties(ctx context.Context, workspaceID uuid.UUID, onlyActive bool, limit, offset int) (*ListPropertiesResult, error) {
	items, total, err := s.repo.ListProperties(ctx, workspaceID, dom.ListPropertiesParams{
		OnlyActive: onlyActive,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}
	return &ListPropertiesResult{Items: items, Total: total}, nil
}

func (s *Service) UpdateProperty(ctx context.Context, in PropertyInput) (*dom.Property, error) {
	p, err := s.repo.GetProperty(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	pt := dom.PropertyType(in.PropertyType)
	if pt == "" {
		pt = p.PropertyType
	}
	p.Name = in.Name
	p.PropertyType = pt
	p.Address = in.Address
	p.City = in.City
	p.State = in.State
	p.ZipCode = in.ZipCode
	p.RegistrationNumber = in.RegistrationNumber
	p.AreaM2 = in.AreaM2
	p.PurchaseDate = in.PurchaseDate
	p.PurchaseValueCents = in.PurchaseValueCents
	p.CurrentValueCents = in.CurrentValueCents
	p.Notes = in.Notes
	if in.Active != nil {
		p.Active = *in.Active
	}
	p.UpdatedAt = time.Now().UTC()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateProperty(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) DeleteProperty(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteProperty(ctx, workspaceID, id)
}

// ─── Tax CRUD ──────────────────────────────────────────────────────────────────

type TaxInput struct {
	WorkspaceID       uuid.UUID
	ID                uuid.UUID // usado no Update
	AssetType         string
	PropertyID        *uuid.UUID
	VehicleID         *uuid.UUID
	TaxType           string
	ReferenceYear     int
	Description       *string
	DueDate           *time.Time
	AmountCents       int64
	PaidCents         int64
	PaidDate          *time.Time
	Status            string
	InstallmentsTotal int
	InstallmentNumber int
	Notes             *string
}

func (s *Service) CreateTax(ctx context.Context, in TaxInput) (*dom.AssetTax, error) {
	now := time.Now().UTC()
	t := &dom.AssetTax{
		ID:                uuid.New(),
		WorkspaceID:       in.WorkspaceID,
		AssetType:         dom.AssetType(in.AssetType),
		PropertyID:        in.PropertyID,
		VehicleID:         in.VehicleID,
		TaxType:           dom.TaxType(in.TaxType),
		ReferenceYear:     in.ReferenceYear,
		Description:       in.Description,
		DueDate:           in.DueDate,
		AmountCents:       in.AmountCents,
		PaidCents:         in.PaidCents,
		PaidDate:          in.PaidDate,
		InstallmentsTotal: in.InstallmentsTotal,
		InstallmentNumber: in.InstallmentNumber,
		Notes:             in.Notes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	// Confirma que o bem referenciado existe no workspace.
	if t.AssetType == dom.AssetProperty && t.PropertyID != nil {
		if _, err := s.repo.GetProperty(ctx, in.WorkspaceID, *t.PropertyID); err != nil {
			return nil, err
		}
	}
	t.Status = resolveStatus(in.Status, t, now)
	if err := s.repo.CreateTax(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) GetTax(ctx context.Context, workspaceID, id uuid.UUID) (*dom.AssetTax, error) {
	return s.repo.GetTax(ctx, workspaceID, id)
}

type ListTaxesResult struct {
	Items []dom.AssetTax
	Total int64
}

func (s *Service) ListTaxes(ctx context.Context, workspaceID uuid.UUID, params dom.ListTaxesParams) (*ListTaxesResult, error) {
	items, total, err := s.repo.ListTaxes(ctx, workspaceID, params)
	if err != nil {
		return nil, err
	}
	return &ListTaxesResult{Items: items, Total: total}, nil
}

func (s *Service) UpdateTax(ctx context.Context, in TaxInput) (*dom.AssetTax, error) {
	t, err := s.repo.GetTax(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if in.AssetType != "" {
		t.AssetType = dom.AssetType(in.AssetType)
	}
	t.PropertyID = in.PropertyID
	t.VehicleID = in.VehicleID
	if in.TaxType != "" {
		t.TaxType = dom.TaxType(in.TaxType)
	}
	t.ReferenceYear = in.ReferenceYear
	t.Description = in.Description
	t.DueDate = in.DueDate
	t.AmountCents = in.AmountCents
	t.PaidCents = in.PaidCents
	t.PaidDate = in.PaidDate
	t.InstallmentsTotal = in.InstallmentsTotal
	t.InstallmentNumber = in.InstallmentNumber
	t.Notes = in.Notes
	t.UpdatedAt = now
	if err := t.Validate(); err != nil {
		return nil, err
	}
	t.Status = resolveStatus(in.Status, t, now)
	if err := s.repo.UpdateTax(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) DeleteTax(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteTax(ctx, workspaceID, id)
}

// PayTaxInput marca (parcial ou totalmente) um imposto como pago.
type PayTaxInput struct {
	WorkspaceID uuid.UUID
	ID          uuid.UUID
	PaidCents   int64
	PaidDate    *time.Time
}

func (s *Service) PayTax(ctx context.Context, in PayTaxInput) (*dom.AssetTax, error) {
	t, err := s.repo.GetTax(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	// paid_cents = 0 → quita integralmente pelo valor devido.
	if in.PaidCents <= 0 {
		t.PaidCents = t.AmountCents
	} else {
		t.PaidCents = in.PaidCents
	}
	if in.PaidDate != nil {
		t.PaidDate = in.PaidDate
	} else {
		today := now
		t.PaidDate = &today
	}
	t.Status = t.ComputeStatus(now)
	t.UpdatedAt = now
	if err := s.repo.UpdateTax(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ─── Overview ──────────────────────────────────────────────────────────────────

func (s *Service) Overview(ctx context.Context, workspaceID uuid.UUID) (*dom.TaxOverview, error) {
	taxes, err := s.repo.ListAllTaxes(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	props, _, err := s.repo.ListProperties(ctx, workspaceID, dom.ListPropertiesParams{OnlyActive: true, Limit: 10000})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	ov := &dom.TaxOverview{}

	// Patrimônio: contagem e valor total (usa current, com fallback para purchase).
	ov.TotalProperties = len(props)
	for i := range props {
		switch {
		case props[i].CurrentValueCents != nil:
			ov.TotalPropertyValue += *props[i].CurrentValueCents
		case props[i].PurchaseValueCents != nil:
			ov.TotalPropertyValue += *props[i].PurchaseValueCents
		}
	}

	// Agregados por ano e por (tipo, ano).
	yearIdx := map[int]*dom.YearTotals{}
	typeYearIdx := map[string]*dom.TaxTypeYearTotals{}
	// amountByTypeYear alimenta o cálculo de inflação YoY.
	amountByTypeYear := map[dom.TaxType]map[int]int64{}

	for i := range taxes {
		t := &taxes[i]

		yt, ok := yearIdx[t.ReferenceYear]
		if !ok {
			yt = &dom.YearTotals{Year: t.ReferenceYear}
			yearIdx[t.ReferenceYear] = yt
		}
		yt.PlannedCents += t.AmountCents
		yt.PaidCents += t.PaidCents

		key := string(t.TaxType) + "|" + strconv.Itoa(t.ReferenceYear)
		tyt, ok := typeYearIdx[key]
		if !ok {
			tyt = &dom.TaxTypeYearTotals{TaxType: t.TaxType, Year: t.ReferenceYear}
			typeYearIdx[key] = tyt
		}
		tyt.PlannedCents += t.AmountCents
		tyt.PaidCents += t.PaidCents

		if amountByTypeYear[t.TaxType] == nil {
			amountByTypeYear[t.TaxType] = map[int]int64{}
		}
		amountByTypeYear[t.TaxType][t.ReferenceYear] += t.AmountCents

		// Próximos vencimentos (90 dias) e vencidos.
		effectiveStatus := t.ComputeStatus(now)
		if effectiveStatus == dom.TaxStatusPaid {
			continue
		}
		if t.DueDate != nil {
			if t.DueDate.Before(now) {
				ov.Overdue = append(ov.Overdue, *t)
			} else if t.DueDate.Before(now.AddDate(0, 0, 90)) {
				ov.Upcoming = append(ov.Upcoming, *t)
			}
		}
	}

	for _, yt := range yearIdx {
		ov.ByYear = append(ov.ByYear, *yt)
	}
	sort.Slice(ov.ByYear, func(i, j int) bool { return ov.ByYear[i].Year < ov.ByYear[j].Year })

	for _, tyt := range typeYearIdx {
		ov.ByTaxTypeYear = append(ov.ByTaxTypeYear, *tyt)
	}
	sort.Slice(ov.ByTaxTypeYear, func(i, j int) bool {
		if ov.ByTaxTypeYear[i].TaxType != ov.ByTaxTypeYear[j].TaxType {
			return ov.ByTaxTypeYear[i].TaxType < ov.ByTaxTypeYear[j].TaxType
		}
		return ov.ByTaxTypeYear[i].Year < ov.ByTaxTypeYear[j].Year
	})

	// Inflação: variação percentual YoY por tipo de imposto (mesmo tipo, ano vs ano-1).
	for taxType, byYear := range amountByTypeYear {
		years := make([]int, 0, len(byYear))
		for y := range byYear {
			years = append(years, y)
		}
		sort.Ints(years)
		for _, y := range years {
			prevAmount, hasPrev := byYear[y-1]
			entry := dom.InflationEntry{
				TaxType:       taxType,
				Year:          y,
				PreviousYear:  y - 1,
				AmountCents:   byYear[y],
				PreviousCents: prevAmount,
			}
			if hasPrev && prevAmount > 0 {
				pct := (float64(byYear[y]) - float64(prevAmount)) / float64(prevAmount) * 100
				entry.VariationPct = &pct
			}
			if hasPrev {
				ov.Inflation = append(ov.Inflation, entry)
			}
		}
	}
	sort.Slice(ov.Inflation, func(i, j int) bool {
		if ov.Inflation[i].TaxType != ov.Inflation[j].TaxType {
			return ov.Inflation[i].TaxType < ov.Inflation[j].TaxType
		}
		return ov.Inflation[i].Year < ov.Inflation[j].Year
	})

	sort.Slice(ov.Upcoming, func(i, j int) bool { return dueBefore(ov.Upcoming[i], ov.Upcoming[j]) })
	sort.Slice(ov.Overdue, func(i, j int) bool { return dueBefore(ov.Overdue[i], ov.Overdue[j]) })

	return ov, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// resolveStatus respeita um status explícito informado pelo cliente; caso vazio,
// deriva o status a partir dos valores/vencimento.
func resolveStatus(explicit string, t *dom.AssetTax, now time.Time) dom.TaxStatus {
	if explicit != "" {
		return dom.TaxStatus(explicit)
	}
	return t.ComputeStatus(now)
}

func dueBefore(a, b dom.AssetTax) bool {
	if a.DueDate == nil {
		return false
	}
	if b.DueDate == nil {
		return true
	}
	return a.DueDate.Before(*b.DueDate)
}
