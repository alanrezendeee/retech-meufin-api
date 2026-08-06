package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// RenegotiationModel mapeia finance_renegotiations.
type RenegotiationModel struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Date               time.Time `gorm:"type:date;not null"`
	Description        string    `gorm:"size:255;not null"`
	SettledAmountCents int64     `gorm:"not null"`
	NewAmountCents     int64     `gorm:"not null"`
	AdjustmentCents    int64     `gorm:"not null"`
	OriginCount        int       `gorm:"not null"`
	NewCount           int       `gorm:"not null"`
	Notes              *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (RenegotiationModel) TableName() string { return "finance_renegotiations" }

type RenegotiationRepository struct {
	db *gorm.DB
}

func NewRenegotiationRepository(db *gorm.DB) *RenegotiationRepository {
	return &RenegotiationRepository{db: db}
}

// Apply grava o evento, encerra as origens e cria as parcelas novas numa
// única transação.
//
// A atomicidade aqui não é preciosismo: se as origens fossem canceladas sem
// que as parcelas novas nascessem, a dívida sumiria dos relatórios; no
// caminho inverso, ela apareceria em dobro.
func (r *RenegotiationRepository) Apply(
	ctx context.Context,
	reneg *dom.Renegotiation,
	originIDs []uuid.UUID,
	newEntries []*dom.FinancialEntry,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := renegotiationToModel(reneg)
		if err := tx.Create(&model).Error; err != nil {
			return mapFinanceErr(err)
		}

		if len(originIDs) > 0 {
			res := tx.Model(&FinancialEntryModel{}).
				Where("workspace_id = ? AND id IN ? AND status = ?",
					reneg.WorkspaceID, originIDs, string(dom.StatusPrevista)).
				Updates(map[string]any{
					"status":           string(dom.StatusCancelada),
					"cancel_reason":    dom.CancelReasonRenegotiation,
					"renegotiation_id": reneg.ID,
					"updated_at":       time.Now().UTC(),
				})
			if res.Error != nil {
				return mapFinanceErr(res.Error)
			}
			// O filtro por status = prevista é a trava de concorrência: se
			// alguém liquidou uma parcela entre a apuração e a confirmação,
			// o saldo renegociado não corresponde mais à realidade.
			if int(res.RowsAffected) != len(originIDs) {
				return &dom.ValidationError{
					Msg: "as cobranças mudaram durante a renegociação (alguma foi paga ou cancelada) — refaça a apuração",
				}
			}
		}

		if len(newEntries) > 0 {
			models := make([]FinancialEntryModel, len(newEntries))
			for i := range newEntries {
				models[i] = financialEntryToModel(newEntries[i])
			}
			if err := tx.Create(&models).Error; err != nil {
				return mapFinanceErr(err)
			}
		}
		return nil
	})
}

func (r *RenegotiationRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Renegotiation, error) {
	var m RenegotiationModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	return modelToRenegotiation(&m), nil
}

func (r *RenegotiationRepository) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]dom.Renegotiation, int64, error) {
	base := r.db.WithContext(ctx).Model(&RenegotiationModel{}).
		Where("workspace_id = ?", workspaceID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	var rows []RenegotiationModel
	if err := base.Order("date DESC, created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapFinanceErr(err)
	}
	out := make([]dom.Renegotiation, len(rows))
	for i := range rows {
		out[i] = *modelToRenegotiation(&rows[i])
	}
	return out, total, nil
}

// ListEntries separa os lançamentos do evento entre origens encerradas e
// parcelas novas — o status distingue os dois papéis.
func (r *RenegotiationRepository) ListEntries(ctx context.Context, workspaceID, renegotiationID uuid.UUID) ([]dom.FinancialEntry, []dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND renegotiation_id = ?", workspaceID, renegotiationID).
		Order("due_date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, nil, mapFinanceErr(err)
	}
	var origins, created []dom.FinancialEntry
	for i := range rows {
		e := *modelToFinancialEntry(&rows[i])
		if e.Status == dom.StatusCancelada {
			origins = append(origins, e)
		} else {
			created = append(created, e)
		}
	}
	return origins, created, nil
}

// --- conversões ---

func renegotiationToModel(r *dom.Renegotiation) RenegotiationModel {
	return RenegotiationModel{
		ID:                 r.ID,
		WorkspaceID:        r.WorkspaceID,
		Date:               r.Date,
		Description:        r.Description,
		SettledAmountCents: r.SettledAmountCents,
		NewAmountCents:     r.NewAmountCents,
		AdjustmentCents:    r.AdjustmentCents,
		OriginCount:        r.OriginCount,
		NewCount:           r.NewCount,
		Notes:              r.Notes,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

func modelToRenegotiation(m *RenegotiationModel) *dom.Renegotiation {
	return &dom.Renegotiation{
		ID:                 m.ID,
		WorkspaceID:        m.WorkspaceID,
		Date:               m.Date,
		Description:        m.Description,
		SettledAmountCents: m.SettledAmountCents,
		NewAmountCents:     m.NewAmountCents,
		AdjustmentCents:    m.AdjustmentCents,
		OriginCount:        m.OriginCount,
		NewCount:           m.NewCount,
		Notes:              m.Notes,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}
