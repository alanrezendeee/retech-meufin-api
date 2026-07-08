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
			"paid_at":            model.PaidAt,
			"paid_amount_cents":  model.PaidAmountCents,
			"payment_method":     model.PaymentMethod,
			"payment_account_id": model.PaymentAccountID,
			"payment_card_id":    model.PaymentCardID,
			"discount_cents":     model.DiscountCents,
			"discount_reason":    model.DiscountReason,
			"residual_of_id":     model.ResidualOfID,
			"purchase_date":      model.PurchaseDate,
			"fiscal_document_id": model.FiscalDocumentID,
			"supplier_id":        model.SupplierID,
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
	if filter.Query != "" {
		base = base.Where("description ILIKE ?", "%"+filter.Query+"%")
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
	// Contas do dia: recortes por vencimento (datas normalizadas para o dia em UTC).
	if filter.DueOn != nil {
		day := filter.DueOn.UTC().Truncate(24 * time.Hour)
		base = base.Where("due_date >= ? AND due_date < ?", day, day.AddDate(0, 0, 1))
	}
	if filter.DueFrom != nil {
		base = base.Where("due_date >= ?", filter.DueFrom.UTC().Truncate(24*time.Hour))
	}
	if filter.DueTo != nil {
		base = base.Where("due_date < ?", filter.DueTo.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1))
	}
	if filter.Overdue {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		base = base.Where("due_date < ? AND status = ?", today, "prevista")
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
	if filter.SupplierID != nil {
		base = base.Where("supplier_id = ?", *filter.SupplierID)
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

// CascadeStatusToChildren propaga o status da fatura pai para os filhos não
// cancelados. Filhos já cancelados individualmente não são reativados.
func (r *FinancialEntryRepository) CascadeStatusToChildren(ctx context.Context, workspaceID, parentID uuid.UUID, status dom.Status, paidAt *time.Time) error {
	updates := map[string]any{
		"status":     string(status),
		"updated_at": time.Now().UTC(),
	}
	if status == dom.StatusRealizada {
		updates["paid_at"] = paidAt
	}
	if status == dom.StatusPrevista {
		// Reabertura: pagamento desfeito, detalhes de liquidação não valem mais.
		updates["paid_at"] = nil
		updates["paid_amount_cents"] = nil
		updates["payment_method"] = nil
		updates["payment_account_id"] = nil
		updates["payment_card_id"] = nil
		updates["discount_cents"] = nil
		updates["discount_reason"] = nil
	}
	res := r.db.WithContext(ctx).Model(&FinancialEntryModel{}).
		Where("parent_id = ? AND workspace_id = ? AND status <> ?", parentID, workspaceID, "cancelada").
		Updates(updates)
	return mapFinanceErr(res.Error)
}

// ListInvoiceInstallments retorna compras parceladas dentro de faturas
// (filhos com parcela preenchida) — insumo da projeção de compromissos.
func (r *FinancialEntryRepository) ListInvoiceInstallments(ctx context.Context, workspaceID uuid.UUID) ([]dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND kind = ? AND parent_id IS NOT NULL AND installment_number IS NOT NULL AND installment_total IS NOT NULL",
			workspaceID, "debit").
		Order("due_date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
}

// ListGroupSiblings retorna os lançamentos não cancelados do grupo de
// recorrência/parcelamento — alvo da edição em série.
func (r *FinancialEntryRepository) ListGroupSiblings(ctx context.Context, workspaceID, groupID uuid.UUID, excludeID uuid.UUID) ([]dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND recurrence_group_id = ? AND status <> ? AND id <> ?",
			workspaceID, groupID, string(dom.StatusCancelada), excludeID).
		Order("due_date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
}

// ListStandaloneInstallments retorna despesas parceladas diretas (sem fatura-pai).
func (r *FinancialEntryRepository) ListStandaloneInstallments(ctx context.Context, workspaceID uuid.UUID) ([]dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND kind = ? AND parent_id IS NULL AND installment_number IS NOT NULL AND installment_total IS NOT NULL",
			workspaceID, "debit").
		Order("due_date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
}

// YearBounds retorna o menor e o maior ano de vencimento do workspace.
func (r *FinancialEntryRepository) YearBounds(ctx context.Context, workspaceID uuid.UUID) (int, int, error) {
	var row struct {
		Min *time.Time
		Max *time.Time
	}
	err := r.db.WithContext(ctx).Model(&FinancialEntryModel{}).
		Select("MIN(due_date) AS min, MAX(due_date) AS max").
		Where("workspace_id = ?", workspaceID).
		Scan(&row).Error
	if err != nil {
		return 0, 0, mapFinanceErr(err)
	}
	if row.Min == nil || row.Max == nil {
		return 0, 0, nil
	}
	return row.Min.Year(), row.Max.Year(), nil
}

// ListResiduals retorna os lançamentos residuais gerados a partir da origem.
func (r *FinancialEntryRepository) ListResiduals(ctx context.Context, workspaceID, originID uuid.UUID) ([]dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND residual_of_id = ?", workspaceID, originID).
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
}

// ListRecurrenceFrontiers retorna a ocorrência mais recente de cada grupo de
// recorrência (todas as workspaces) — insumo do extensor rolling.
func (r *FinancialEntryRepository) ListRecurrenceFrontiers(ctx context.Context) ([]dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT ON (recurrence_group_id) *
		     FROM financial_entries
		     WHERE recurrence <> 'none' AND recurrence_group_id IS NOT NULL AND deleted_at IS NULL
		     ORDER BY recurrence_group_id, due_date DESC`).
		Scan(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.FinancialEntry, len(rows))
	for i := range rows {
		out[i] = *modelToFinancialEntry(&rows[i])
	}
	return out, nil
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
		PaidAt:            e.PaidAt,
		PaidAmountCents:   e.PaidAmountCents,
		PaymentMethod:     paymentMethodToString(e.PaymentMethod),
		PaymentAccountID:  e.PaymentAccountID,
		PaymentCardID:     e.PaymentCardID,
		DiscountCents:     e.DiscountCents,
		DiscountReason:    e.DiscountReason,
		ResidualOfID:      e.ResidualOfID,
		PurchaseDate:      e.PurchaseDate,
		FiscalDocumentID:  e.FiscalDocumentID,
		SupplierID:        e.SupplierID,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

func paymentMethodToString(m *dom.PaymentMethod) *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

func stringToPaymentMethod(s *string) *dom.PaymentMethod {
	if s == nil {
		return nil
	}
	m := dom.PaymentMethod(*s)
	return &m
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
		PaidAt:            m.PaidAt,
		PaidAmountCents:   m.PaidAmountCents,
		PaymentMethod:     stringToPaymentMethod(m.PaymentMethod),
		PaymentAccountID:  m.PaymentAccountID,
		PaymentCardID:     m.PaymentCardID,
		DiscountCents:     m.DiscountCents,
		DiscountReason:    m.DiscountReason,
		ResidualOfID:      m.ResidualOfID,
		PurchaseDate:      m.PurchaseDate,
		FiscalDocumentID:  m.FiscalDocumentID,
		SupplierID:        m.SupplierID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
