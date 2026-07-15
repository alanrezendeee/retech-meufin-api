package warranty

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/warranty"
)

// Service orquestra os casos de uso do módulo de garantias de bens.
type Service struct {
	repo dom.Repository
}

func NewService(repo dom.Repository) *Service {
	return &Service{repo: repo}
}

// defaults CDC / fabricante quando o cliente não informa.
const (
	defaultLegalWarrantyDays         = 90
	defaultContractualWarrantyMonths = 12
)

// ─── Inputs ───────────────────────────────────────────────────────────────────

type CreateInput struct {
	WorkspaceID               uuid.UUID
	ItemName                  string
	Category                  string
	Brand                     *string
	Model                     *string
	SerialNumber              *string
	Store                     *string
	SupplierName              *string
	PurchaseDate              time.Time
	PriceCents                *int64
	InvoiceNumber             *string
	EntryID                   *uuid.UUID
	FiscalItemID              *uuid.UUID
	LegalWarrantyDays         *int
	ContractualWarrantyMonths *int
	ExtendedWarrantyMonths    *int
	ExtendedProvider          *string
	ExtendedCostCents         *int64
	CoverageNotes             *string
	Notes                     *string
	Active                    *bool
}

type UpdateInput struct {
	WorkspaceID               uuid.UUID
	ID                        uuid.UUID
	ItemName                  string
	Category                  string
	Brand                     *string
	Model                     *string
	SerialNumber              *string
	Store                     *string
	SupplierName              *string
	PurchaseDate              time.Time
	PriceCents                *int64
	InvoiceNumber             *string
	EntryID                   *uuid.UUID
	FiscalItemID              *uuid.UUID
	LegalWarrantyDays         *int
	ContractualWarrantyMonths *int
	ExtendedWarrantyMonths    *int
	ExtendedProvider          *string
	ExtendedCostCents         *int64
	CoverageNotes             *string
	Notes                     *string
	Active                    *bool
}

// ─── Warranties ───────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, in CreateInput) (*dom.Warranty, error) {
	now := time.Now().UTC()
	w := &dom.Warranty{
		ID:                        uuid.New(),
		WorkspaceID:               in.WorkspaceID,
		ItemName:                  strings.TrimSpace(in.ItemName),
		Category:                  dom.Category(strings.TrimSpace(strings.ToLower(in.Category))),
		Brand:                     in.Brand,
		Model:                     in.Model,
		SerialNumber:              in.SerialNumber,
		Store:                     in.Store,
		SupplierName:              in.SupplierName,
		PurchaseDate:              in.PurchaseDate,
		PriceCents:                in.PriceCents,
		InvoiceNumber:             in.InvoiceNumber,
		EntryID:                   in.EntryID,
		FiscalItemID:              in.FiscalItemID,
		LegalWarrantyDays:         valueOr(in.LegalWarrantyDays, defaultLegalWarrantyDays),
		ContractualWarrantyMonths: valueOr(in.ContractualWarrantyMonths, defaultContractualWarrantyMonths),
		ExtendedWarrantyMonths:    valueOr(in.ExtendedWarrantyMonths, 0),
		ExtendedProvider:          in.ExtendedProvider,
		ExtendedCostCents:         valueOr64(in.ExtendedCostCents, 0),
		CoverageNotes:             in.CoverageNotes,
		Notes:                     in.Notes,
		Active:                    boolOr(in.Active, true),
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if w.Category == "" {
		w.Category = dom.CategoryOutros
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Warranty, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListResult struct {
	Items []dom.Warranty
	Total int64
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, p dom.ListParams) (*ListResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, p)
	if err != nil {
		return nil, err
	}
	// O status é campo calculado: quando filtrado, aplicamos em memória sobre o
	// conjunto já filtrado por categoria/busca.
	if st := strings.TrimSpace(p.Status); st != "" {
		ref := time.Now().UTC()
		filtered := make([]dom.Warranty, 0, len(items))
		for i := range items {
			if string(items[i].StatusAt(ref)) == st {
				filtered = append(filtered, items[i])
			}
		}
		return &ListResult{Items: filtered, Total: int64(len(filtered))}, nil
	}
	return &ListResult{Items: items, Total: total}, nil
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (*dom.Warranty, error) {
	existing, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	existing.ItemName = strings.TrimSpace(in.ItemName)
	existing.Category = dom.Category(strings.TrimSpace(strings.ToLower(in.Category)))
	if existing.Category == "" {
		existing.Category = dom.CategoryOutros
	}
	existing.Brand = in.Brand
	existing.Model = in.Model
	existing.SerialNumber = in.SerialNumber
	existing.Store = in.Store
	existing.SupplierName = in.SupplierName
	existing.PurchaseDate = in.PurchaseDate
	existing.PriceCents = in.PriceCents
	existing.InvoiceNumber = in.InvoiceNumber
	existing.EntryID = in.EntryID
	existing.FiscalItemID = in.FiscalItemID
	existing.LegalWarrantyDays = valueOr(in.LegalWarrantyDays, defaultLegalWarrantyDays)
	existing.ContractualWarrantyMonths = valueOr(in.ContractualWarrantyMonths, defaultContractualWarrantyMonths)
	existing.ExtendedWarrantyMonths = valueOr(in.ExtendedWarrantyMonths, 0)
	existing.ExtendedProvider = in.ExtendedProvider
	existing.ExtendedCostCents = valueOr64(in.ExtendedCostCents, 0)
	existing.CoverageNotes = in.CoverageNotes
	existing.Notes = in.Notes
	existing.Active = boolOr(in.Active, existing.Active)
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.Delete(ctx, workspaceID, id)
}

// ─── Summary ──────────────────────────────────────────────────────────────────

func (s *Service) Summary(ctx context.Context, workspaceID uuid.UUID) (*dom.Summary, error) {
	items, err := s.repo.ListActive(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ref := time.Now().UTC()
	sum := &dom.Summary{}
	byCategory := map[dom.Category]int{}
	expiringSoon := make([]dom.ExpiringItem, 0)

	for i := range items {
		w := &items[i]
		sum.TotalActive++
		byCategory[w.Category]++
		status := w.StatusAt(ref)
		days := w.DaysRemaining(ref)

		if status != dom.StatusExpirada {
			if w.PriceCents != nil {
				sum.TotalCoveredCents += *w.PriceCents
			}
		}
		if days >= 0 {
			if days <= 30 {
				sum.ExpiringIn30Count++
			}
			if days <= 60 {
				sum.ExpiringIn60Count++
			}
			if days <= 90 {
				sum.ExpiringIn90Count++
			}
			if days <= 90 {
				expiringSoon = append(expiringSoon, dom.ExpiringItem{
					ID:            w.ID,
					ItemName:      w.ItemName,
					Category:      w.Category,
					ExpiresAt:     w.ExpiresAt(),
					DaysRemaining: days,
				})
			}
		} else if w.ExpiresAt().Year() == ref.Year() {
			sum.ExpiredThisYear++
		}
	}

	sort.Slice(expiringSoon, func(a, b int) bool {
		return expiringSoon[a].DaysRemaining < expiringSoon[b].DaysRemaining
	})
	if len(expiringSoon) > 10 {
		expiringSoon = expiringSoon[:10]
	}
	sum.ExpiringSoon = expiringSoon

	cats := make([]dom.CategoryCount, 0, len(byCategory))
	for c, n := range byCategory {
		cats = append(cats, dom.CategoryCount{Category: c, Count: n})
	}
	sort.Slice(cats, func(a, b int) bool { return cats[a].Count > cats[b].Count })
	sum.ByCategory = cats

	return sum, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func valueOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func valueOr64(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
