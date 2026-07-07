package persistence

import (
	"context"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

type SupplierRepository struct {
	db *gorm.DB
}

func NewSupplierRepository(db *gorm.DB) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(ctx context.Context, s *dom.Supplier) error {
	model := supplierToModel(s)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *SupplierRepository) Update(ctx context.Context, s *dom.Supplier) error {
	var billingStr *string
	if s.DefaultBillingType != nil {
		v := string(*s.DefaultBillingType)
		billingStr = &v
	}
	res := r.db.WithContext(ctx).Model(&SupplierModel{}).
		Where("id = ? AND workspace_id = ?", s.ID, s.WorkspaceID).
		Updates(map[string]any{
			"name":                 s.Name,
			"category":             string(s.Category),
			"default_billing_type": billingStr,
			"pix_key":              s.PixKey,
			"pix_key_holder":       s.PixKeyHolder,
			"bank_name":            s.BankName,
			"bank_agency":          s.BankAgency,
			"bank_account":         s.BankAccount,
			"bank_account_type":    s.BankAccountType,
			"notes":                s.Notes,
			"active":               s.Active,
			"updated_at":           s.UpdatedAt,
		})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *SupplierRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&SupplierModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *SupplierRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Supplier, error) {
	var m SupplierModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND (workspace_id = ? OR workspace_id IS NULL)", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToSupplier(&m), nil
}

func (r *SupplierRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.SupplierFilter, limit, offset int) ([]dom.Supplier, int64, error) {
	base := r.db.WithContext(ctx).Model(&SupplierModel{}).
		Where("workspace_id = ? OR workspace_id IS NULL", workspaceID)

	if filter.Query != "" {
		base = base.Where("name ILIKE ?", "%"+filter.Query+"%")
	}
	if filter.Category != "" {
		base = base.Where("category = ?", filter.Category)
	}
	if filter.Active != nil {
		base = base.Where("active = ?", *filter.Active)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []SupplierModel
	if err := base.Order("workspace_id ASC NULLS FIRST, name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.Supplier, len(rows))
	for i := range rows {
		out[i] = *modelToSupplier(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func supplierToModel(s *dom.Supplier) SupplierModel {
	var billingStr *string
	if s.DefaultBillingType != nil {
		v := string(*s.DefaultBillingType)
		billingStr = &v
	}
	return SupplierModel{
		ID:                 s.ID,
		WorkspaceID:        s.WorkspaceID,
		Name:               s.Name,
		Category:           string(s.Category),
		DefaultBillingType: billingStr,
		PixKey:             s.PixKey,
		PixKeyHolder:       s.PixKeyHolder,
		BankName:           s.BankName,
		BankAgency:         s.BankAgency,
		BankAccount:        s.BankAccount,
		BankAccountType:    s.BankAccountType,
		Notes:              s.Notes,
		Active:             s.Active,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}

func modelToSupplier(m *SupplierModel) *dom.Supplier {
	var billing *dom.SupplierBillingType
	if m.DefaultBillingType != nil {
		v := dom.SupplierBillingType(*m.DefaultBillingType)
		billing = &v
	}
	return &dom.Supplier{
		ID:                 m.ID,
		WorkspaceID:        m.WorkspaceID,
		Name:               m.Name,
		Category:           dom.SupplierCategory(m.Category),
		DefaultBillingType: billing,
		PixKey:             m.PixKey,
		PixKeyHolder:       m.PixKeyHolder,
		BankName:           m.BankName,
		BankAgency:         m.BankAgency,
		BankAccount:        m.BankAccount,
		BankAccountType:    m.BankAccountType,
		Notes:              m.Notes,
		Active:             m.Active,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}
