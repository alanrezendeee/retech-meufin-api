package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

type FinancialEntryRepository struct {
	db *gorm.DB
}

func NewFinancialEntryRepository(db *gorm.DB) *FinancialEntryRepository {
	return &FinancialEntryRepository{db: db}
}

func (r *FinancialEntryRepository) Create(ctx context.Context, e *dom.FinancialEntry) error {
	model := financialEntryToModel(e)
	return mapFinanceErr(r.db.WithContext(ctx).Create(&model).Error)
}

// CreateBatch insere todas as ocorrências em uma única transação.
func (r *FinancialEntryRepository) CreateBatch(ctx context.Context, es []*dom.FinancialEntry) error {
	if len(es) == 0 {
		return nil
	}
	models := make([]FinancialEntryModel, len(es))
	for i := range es {
		models[i] = financialEntryToModel(es[i])
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models).Error; err != nil {
			return mapFinanceErr(err)
		}
		return nil
	})
}

func (r *FinancialEntryRepository) Update(ctx context.Context, e *dom.FinancialEntry) error {
	model := financialEntryToModel(e)
	res := r.db.WithContext(ctx).Model(&FinancialEntryModel{}).
		Where("id = ? AND workspace_id = ?", e.ID, e.WorkspaceID).
		Updates(map[string]any{
			"kind":               model.Kind,
			"status":             model.Status,
			"amount_cents":       model.AmountCents,
			"due_date":           model.DueDate,
			"family_member_id":   model.FamilyMemberID,
			"source_id":          model.SourceID,
			"type":               model.Type,
			"description":        model.Description,
			"recurrence":         model.Recurrence,
			"card_id":            model.CardID,
			"parent_id":          model.ParentID,
			"installment_number": model.InstallmentNumber,
			"installment_total":  model.InstallmentTotal,
			"notes":              model.Notes,
			"updated_at":         model.UpdatedAt,
		})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinancialEntryRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Delete(&FinancialEntryModel{})
	if res.Error != nil {
		return mapFinanceErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *FinancialEntryRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	var m FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToFinancialEntry(&m), nil
}

func (r *FinancialEntryRepository) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FinancialEntryFilter, limit, offset int) ([]dom.FinancialEntry, int64, error) {
	base := r.db.WithContext(ctx).Model(&FinancialEntryModel{}).Where("workspace_id = ?", workspaceID)

	if filter.Kind != nil && *filter.Kind != "" {
		base = base.Where("kind = ?", *filter.Kind)
	}
	if filter.Status != nil && *filter.Status != "" {
		base = base.Where("status = ?", *filter.Status)
	}
	if filter.FamilyMemberID != nil {
		base = base.Where("family_member_id = ?", *filter.FamilyMemberID)
	}
	if filter.Type != nil && *filter.Type != "" {
		base = base.Where("type = ?", *filter.Type)
	}
	if filter.CardID != nil {
		base = base.Where("card_id = ?", *filter.CardID)
	}
	if filter.ParentID != nil {
		base = base.Where("parent_id = ?", *filter.ParentID)
	}
	if filter.TopLevelOnly {
		base = base.Where("parent_id IS NULL")
	}
	// Filtro por exercício via range de datas em due_date.
	if filter.Year != nil {
		loc := time.UTC
		if filter.Month != nil {
			start := time.Date(*filter.Year, time.Month(*filter.Month), 1, 0, 0, 0, 0, loc)
			end := start.AddDate(0, 1, 0)
			base = base.Where("due_date >= ? AND due_date < ?", start, end)
		} else {
			start := time.Date(*filter.Year, time.January, 1, 0, 0, 0, 0, loc)
			end := start.AddDate(1, 0, 0)
			base = base.Where("due_date >= ? AND due_date < ?", start, end)
		}
	} else if filter.Month != nil {
		base = base.Where("EXTRACT(MONTH FROM due_date) = ?", *filter.Month)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []FinancialEntryModel
	if err := base.Order("due_date ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, total, nil
}

// --- conversões ---

func financialEntryToModel(e *dom.FinancialEntry) FinancialEntryModel {
	return FinancialEntryModel{
		ID:                e.ID,
		WorkspaceID:       e.WorkspaceID,
		Kind:              string(e.Kind),
		Status:            string(e.Status),
		AmountCents:       e.AmountCents,
		DueDate:           e.DueDate,
		FamilyMemberID:    e.FamilyMemberID,
		SourceID:          e.SourceID,
		Type:              e.Type,
		Description:       e.Description,
		Recurrence:        string(e.Recurrence),
		RecurrenceGroupID: e.RecurrenceGroupID,
		CardID:            e.CardID,
		ParentID:          e.ParentID,
		InstallmentNumber: e.InstallmentNumber,
		InstallmentTotal:  e.InstallmentTotal,
		Notes:             e.Notes,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

func modelToFinancialEntry(m *FinancialEntryModel) *dom.FinancialEntry {
	return &dom.FinancialEntry{
		ID:                m.ID,
		WorkspaceID:       m.WorkspaceID,
		Kind:              dom.Kind(m.Kind),
		Status:            dom.Status(m.Status),
		AmountCents:       m.AmountCents,
		DueDate:           m.DueDate,
		FamilyMemberID:    m.FamilyMemberID,
		SourceID:          m.SourceID,
		Type:              m.Type,
		Description:       m.Description,
		Recurrence:        dom.Recurrence(m.Recurrence),
		RecurrenceGroupID: m.RecurrenceGroupID,
		CardID:            m.CardID,
		ParentID:          m.ParentID,
		InstallmentNumber: m.InstallmentNumber,
		InstallmentTotal:  m.InstallmentTotal,
		Notes:             m.Notes,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
