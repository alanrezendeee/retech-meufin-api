package finance

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type SupplierService struct {
	repo dom.SupplierRepository
}

func NewSupplierService(repo dom.SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}

type CreateSupplierInput struct {
	WorkspaceID        uuid.UUID
	Name               string
	Category           string
	DefaultBillingType *string
	PixKey             *string
	PixKeyHolder       *string
	BankName           *string
	BankAgency         *string
	BankAccount        *string
	BankAccountType    *string
	Notes              *string
	Active             *bool
}

type UpdateSupplierInput struct {
	WorkspaceID        uuid.UUID
	ID                 uuid.UUID
	Name               string
	Category           string
	DefaultBillingType *string
	PixKey             *string
	PixKeyHolder       *string
	BankName           *string
	BankAgency         *string
	BankAccount        *string
	BankAccountType    *string
	Notes              *string
	Active             *bool
}

func (s *SupplierService) Create(ctx context.Context, in CreateSupplierInput) (*dom.Supplier, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	var billing *dom.SupplierBillingType
	if in.DefaultBillingType != nil {
		v := dom.SupplierBillingType(*in.DefaultBillingType)
		billing = &v
	}
	ws := in.WorkspaceID
	sup := &dom.Supplier{
		ID:                 uuid.New(),
		WorkspaceID:        &ws,
		Name:               in.Name,
		Category:           dom.SupplierCategory(in.Category),
		DefaultBillingType: billing,
		PixKey:             in.PixKey,
		PixKeyHolder:       in.PixKeyHolder,
		BankName:           in.BankName,
		BankAgency:         in.BankAgency,
		BankAccount:        in.BankAccount,
		BankAccountType:    in.BankAccountType,
		Notes:              in.Notes,
		Active:             active,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := sup.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, sup); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *SupplierService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Supplier, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListSuppliersResult struct {
	Items []dom.Supplier
	Total int64
}

func (s *SupplierService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.SupplierFilter, limit, offset int) (*ListSuppliersResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListSuppliersResult{Items: items, Total: total}, nil
}

func (s *SupplierService) Update(ctx context.Context, in UpdateSupplierInput) (*dom.Supplier, error) {
	sup, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	if sup.IsGlobal() {
		return nil, &dom.ValidationError{Msg: "fornecedores globais não podem ser editados"}
	}
	sup.Name = in.Name
	sup.Category = dom.SupplierCategory(in.Category)
	if in.DefaultBillingType != nil {
		v := dom.SupplierBillingType(*in.DefaultBillingType)
		sup.DefaultBillingType = &v
	} else {
		sup.DefaultBillingType = nil
	}
	sup.PixKey = in.PixKey
	sup.PixKeyHolder = in.PixKeyHolder
	sup.BankName = in.BankName
	sup.BankAgency = in.BankAgency
	sup.BankAccount = in.BankAccount
	sup.BankAccountType = in.BankAccountType
	sup.Notes = in.Notes
	if in.Active != nil {
		sup.Active = *in.Active
	}
	sup.UpdatedAt = time.Now().UTC()
	if err := sup.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, sup); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *SupplierService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	sup, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if sup.IsGlobal() {
		return &dom.ValidationError{Msg: "fornecedores globais não podem ser removidos"}
	}
	return s.repo.SoftDelete(ctx, workspaceID, id)
}
