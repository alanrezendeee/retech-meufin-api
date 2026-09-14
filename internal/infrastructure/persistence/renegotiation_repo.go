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
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	Kind                 string     `gorm:"size:20;not null;default:renegociacao"`
	Date                 time.Time  `gorm:"type:date;not null"`
	Description          string     `gorm:"size:255;not null"`
	SettledAmountCents   int64      `gorm:"not null"`
	NewAmountCents       int64      `gorm:"not null"`
	AdjustmentCents      int64      `gorm:"not null"`
	OriginCount          int        `gorm:"not null"`
	NewCount             int        `gorm:"not null"`
	OriginGroupID        *uuid.UUID `gorm:"type:uuid"`
	NewGroupID           *uuid.UUID `gorm:"type:uuid"`
	Notes                *string
	PayoffCents          *int64     `gorm:"column:payoff_cents"`
	Payer                *string    `gorm:"column:payer;size:20"`
	PayoffEntryID        *uuid.UUID `gorm:"column:payoff_entry_id;type:uuid"`
	AssetType            *string    `gorm:"column:asset_type;size:20"`
	AssetID              *uuid.UUID `gorm:"column:asset_id;type:uuid"`
	NewAssetID           *uuid.UUID `gorm:"column:new_asset_id;type:uuid"`
	TradeInCents         *int64     `gorm:"column:trade_in_cents"`
	TradeInEntryID       *uuid.UUID `gorm:"column:trade_in_entry_id;type:uuid"`
	CashDownpaymentCents *int64     `gorm:"column:cash_downpayment_cents"`
	DownpaymentEntryID   *uuid.UUID `gorm:"column:downpayment_entry_id;type:uuid"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (RenegotiationModel) TableName() string { return "finance_renegotiations" }

type RenegotiationRepository struct {
	db *gorm.DB
}

func NewRenegotiationRepository(db *gorm.DB) *RenegotiationRepository {
	return &RenegotiationRepository{db: db}
}

// Apply grava o evento, encerra as origens, cria os lançamentos novos e
// atualiza o bem (na troca) numa única transação.
//
// A atomicidade aqui não é preciosismo: se as origens fossem canceladas sem
// que a quitação/parcelas novas nascessem, a dívida sumiria dos relatórios;
// no caminho inverso, ela apareceria em dobro. E um veículo marcado como
// vendido sem a receita da venda deixaria o patrimônio furado.
//
// Os lançamentos novos referenciam o evento (renegotiation_id) e o evento
// referencia alguns deles (payoff_entry_id etc.). Para não depender de FK
// diferida, o evento nasce sem esses ponteiros e recebe-os num UPDATE ao fim
// da mesma transação.
func (r *RenegotiationRepository) Apply(ctx context.Context, in dom.ApplyInput) error {
	reneg := in.Event
	cancelReason := in.CancelReason
	if cancelReason == "" {
		cancelReason = dom.CancelReasonRenegotiation
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := renegotiationToModel(reneg)
		payoffEntry, tradeInEntry, downpaymentEntry := model.PayoffEntryID, model.TradeInEntryID, model.DownpaymentEntryID
		model.PayoffEntryID, model.TradeInEntryID, model.DownpaymentEntryID = nil, nil, nil
		if err := tx.Create(&model).Error; err != nil {
			return mapFinanceErr(err)
		}

		if len(in.OriginIDs) > 0 {
			res := tx.Model(&FinancialEntryModel{}).
				Where("workspace_id = ? AND id IN ? AND status = ?",
					reneg.WorkspaceID, in.OriginIDs, string(dom.StatusPrevista)).
				// Só o "encerrado por" muda: o "criado por" (renegotiation_id)
				// fica, para o evento anterior continuar enxergando as
				// parcelas que criou mesmo depois de encerradas de novo.
				Updates(map[string]any{
					"status":                      string(dom.StatusCancelada),
					"cancel_reason":               cancelReason,
					"settled_by_renegotiation_id": reneg.ID,
					"updated_at":                  time.Now().UTC(),
				})
			if res.Error != nil {
				return mapFinanceErr(res.Error)
			}
			// O filtro por status = prevista é a trava de concorrência: se
			// alguém liquidou uma parcela entre a apuração e a confirmação,
			// o saldo apurado não corresponde mais à realidade.
			if int(res.RowsAffected) != len(in.OriginIDs) {
				return &dom.ValidationError{
					Msg: "as cobranças mudaram durante a operação (alguma foi paga ou cancelada) — refaça a apuração",
				}
			}
		}

		if len(in.NewEntries) > 0 {
			models := make([]FinancialEntryModel, len(in.NewEntries))
			for i := range in.NewEntries {
				models[i] = financialEntryToModel(in.NewEntries[i])
			}
			if err := tx.Create(&models).Error; err != nil {
				return mapFinanceErr(err)
			}
		}

		if payoffEntry != nil || tradeInEntry != nil || downpaymentEntry != nil {
			if err := tx.Model(&RenegotiationModel{}).Where("id = ?", model.ID).Updates(map[string]any{
				"payoff_entry_id":      payoffEntry,
				"trade_in_entry_id":    tradeInEntry,
				"downpayment_entry_id": downpaymentEntry,
			}).Error; err != nil {
				return mapFinanceErr(err)
			}
		}

		if in.Sale != nil {
			if err := applyAssetSale(tx, reneg.WorkspaceID, in.Sale); err != nil {
				return err
			}
		}
		if in.Acquisition != nil {
			if err := applyAssetAcquisition(tx, reneg.WorkspaceID, in.Acquisition); err != nil {
				return err
			}
		}
		return nil
	})
}

// applyAssetSale marca o bem como vendido. O cadastro de veículos guarda
// dinheiro em reais (NUMERIC), não em centavos — a conversão é feita aqui,
// na fronteira, e em nenhum outro lugar.
func applyAssetSale(tx *gorm.DB, workspaceID uuid.UUID, sale *dom.AssetSale) error {
	switch sale.AssetType {
	case dom.AssetVehicle:
		price := float64(sale.PriceCents) / 100
		res := tx.Model(&VehicleModel{}).
			Where("id = ? AND workspace_id = ?", sale.AssetID.String(), workspaceID.String()).
			Updates(map[string]any{
				"status":     "sold",
				"sold_at":    sale.SoldAt,
				"sold_price": price,
				"updated_at": time.Now().UTC(),
			})
		if res.Error != nil {
			return mapFinanceErr(res.Error)
		}
		if res.RowsAffected == 0 {
			return &dom.ValidationError{Msg: "veículo antigo não encontrado"}
		}
		return nil
	default:
		return &dom.ValidationError{Msg: "tipo de bem não suportado na troca"}
	}
}

// applyAssetAcquisition preenche data/preço de aquisição do bem novo quando
// ainda vazios — o usuário pode ter cadastrado o bem já com esses dados.
func applyAssetAcquisition(tx *gorm.DB, workspaceID uuid.UUID, acq *dom.AssetAcquisition) error {
	switch acq.AssetType {
	case dom.AssetVehicle:
		var v VehicleModel
		err := tx.Where("id = ? AND workspace_id = ?", acq.AssetID.String(), workspaceID.String()).First(&v).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return &dom.ValidationError{Msg: "veículo novo não encontrado"}
			}
			return mapFinanceErr(err)
		}
		updates := map[string]any{"updated_at": time.Now().UTC()}
		if v.AcquisitionDate == nil {
			updates["acquisition_date"] = acq.AcquiredAt
		}
		if v.AcquisitionPrice == nil && acq.PriceCents != nil {
			updates["acquisition_price"] = float64(*acq.PriceCents) / 100
		}
		if len(updates) == 1 {
			return nil
		}
		if err := tx.Model(&VehicleModel{}).Where("id = ?", v.ID).Updates(updates).Error; err != nil {
			return mapFinanceErr(err)
		}
		return nil
	default:
		return &dom.ValidationError{Msg: "tipo de bem não suportado na troca"}
	}
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

// ListEntries separa os lançamentos do evento entre origens encerradas
// (settled_by_renegotiation_id) e criados por ele (renegotiation_id).
func (r *RenegotiationRepository) ListEntries(ctx context.Context, workspaceID, renegotiationID uuid.UUID) ([]dom.FinancialEntry, []dom.FinancialEntry, error) {
	var rows []FinancialEntryModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND (renegotiation_id = ? OR settled_by_renegotiation_id = ?)",
			workspaceID, renegotiationID, renegotiationID).
		Order("due_date ASC").
		Find(&rows).Error
	if err != nil {
		return nil, nil, mapFinanceErr(err)
	}
	var origins, created []dom.FinancialEntry
	for i := range rows {
		e := *modelToFinancialEntry(&rows[i])
		if e.SettledByRenegotiationID != nil && *e.SettledByRenegotiationID == renegotiationID {
			origins = append(origins, e)
		}
		if e.RenegotiationID != nil && *e.RenegotiationID == renegotiationID {
			created = append(created, e)
		}
	}
	return origins, created, nil
}

func (r *RenegotiationRepository) FindByNewGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*dom.Renegotiation, error) {
	return r.findByGroup(ctx, workspaceID, "new_group_id", groupID)
}

func (r *RenegotiationRepository) FindByOriginGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*dom.Renegotiation, error) {
	return r.findByGroup(ctx, workspaceID, "origin_group_id", groupID)
}

func (r *RenegotiationRepository) findByGroup(ctx context.Context, workspaceID uuid.UUID, column string, groupID uuid.UUID) (*dom.Renegotiation, error) {
	var m RenegotiationModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND "+column+" = ?", workspaceID, groupID).
		Order("date DESC, created_at DESC").
		First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, mapFinanceErr(err)
	}
	return modelToRenegotiation(&m), nil
}

func (r *RenegotiationRepository) ListByAsset(ctx context.Context, workspaceID uuid.UUID, assetType dom.AssetType, assetID uuid.UUID) ([]dom.Renegotiation, error) {
	var rows []RenegotiationModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND ((asset_type = ? AND asset_id = ?) OR new_asset_id = ?)",
			workspaceID, string(assetType), assetID, assetID).
		Order("date DESC, created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	out := make([]dom.Renegotiation, len(rows))
	for i := range rows {
		out[i] = *modelToRenegotiation(&rows[i])
	}
	return out, nil
}

// --- conversões ---

func renegotiationToModel(r *dom.Renegotiation) RenegotiationModel {
	kind := string(r.Kind)
	if kind == "" {
		kind = string(dom.KindRenegotiation)
	}
	var payer *string
	if r.Payer != nil {
		p := string(*r.Payer)
		payer = &p
	}
	return RenegotiationModel{
		ID:                   r.ID,
		WorkspaceID:          r.WorkspaceID,
		Kind:                 kind,
		Date:                 r.Date,
		Description:          r.Description,
		SettledAmountCents:   r.SettledAmountCents,
		NewAmountCents:       r.NewAmountCents,
		AdjustmentCents:      r.AdjustmentCents,
		OriginCount:          r.OriginCount,
		NewCount:             r.NewCount,
		OriginGroupID:        r.OriginGroupID,
		NewGroupID:           r.NewGroupID,
		Notes:                r.Notes,
		PayoffCents:          r.PayoffCents,
		Payer:                payer,
		PayoffEntryID:        r.PayoffEntryID,
		AssetType:            assetTypeToString(r.AssetType),
		AssetID:              r.AssetID,
		NewAssetID:           r.NewAssetID,
		TradeInCents:         r.TradeInCents,
		TradeInEntryID:       r.TradeInEntryID,
		CashDownpaymentCents: r.CashDownpaymentCents,
		DownpaymentEntryID:   r.DownpaymentEntryID,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func modelToRenegotiation(m *RenegotiationModel) *dom.Renegotiation {
	var payer *dom.Payer
	if m.Payer != nil && *m.Payer != "" {
		p := dom.Payer(*m.Payer)
		payer = &p
	}
	kind := dom.RenegotiationKind(m.Kind)
	if kind == "" {
		kind = dom.KindRenegotiation
	}
	return &dom.Renegotiation{
		ID:                   m.ID,
		WorkspaceID:          m.WorkspaceID,
		Kind:                 kind,
		Date:                 m.Date,
		Description:          m.Description,
		SettledAmountCents:   m.SettledAmountCents,
		NewAmountCents:       m.NewAmountCents,
		AdjustmentCents:      m.AdjustmentCents,
		OriginCount:          m.OriginCount,
		NewCount:             m.NewCount,
		OriginGroupID:        m.OriginGroupID,
		NewGroupID:           m.NewGroupID,
		Notes:                m.Notes,
		PayoffCents:          m.PayoffCents,
		Payer:                payer,
		PayoffEntryID:        m.PayoffEntryID,
		AssetType:            stringToAssetType(m.AssetType),
		AssetID:              m.AssetID,
		NewAssetID:           m.NewAssetID,
		TradeInCents:         m.TradeInCents,
		TradeInEntryID:       m.TradeInEntryID,
		CashDownpaymentCents: m.CashDownpaymentCents,
		DownpaymentEntryID:   m.DownpaymentEntryID,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}
