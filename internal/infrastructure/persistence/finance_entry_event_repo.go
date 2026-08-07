package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"gorm.io/gorm"
)

// FinanceEntryEventModel mapeia finance_entry_events (append-only).
type FinanceEntryEventModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID     uuid.UUID `gorm:"type:uuid;not null;index"`
	EntryID         uuid.UUID `gorm:"type:uuid;not null"`
	Event           string    `gorm:"size:30;not null"`
	FromStatus      *string   `gorm:"size:15"`
	ToStatus        *string   `gorm:"size:15"`
	PaidAt          *time.Time
	PaidAmountCents *int64
	CancelReason    *string `gorm:"size:30"`
	OldDueDate      *time.Time
	NewDueDate      *time.Time
	ActorUserID     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
}

func (FinanceEntryEventModel) TableName() string { return "finance_entry_events" }

type FinanceEntryEventRepository struct {
	db *gorm.DB
}

func NewFinanceEntryEventRepository(db *gorm.DB) *FinanceEntryEventRepository {
	return &FinanceEntryEventRepository{db: db}
}

func (r *FinanceEntryEventRepository) Create(ctx context.Context, ev *dom.EntryEvent) error {
	statusStr := func(s *dom.Status) *string {
		if s == nil {
			return nil
		}
		v := string(*s)
		return &v
	}
	m := FinanceEntryEventModel{
		ID:              ev.ID,
		WorkspaceID:     ev.WorkspaceID,
		EntryID:         ev.EntryID,
		Event:           string(ev.Event),
		FromStatus:      statusStr(ev.FromStatus),
		ToStatus:        statusStr(ev.ToStatus),
		PaidAt:          ev.PaidAt,
		PaidAmountCents: ev.PaidAmountCents,
		CancelReason:    ev.CancelReason,
		OldDueDate:      ev.OldDueDate,
		NewDueDate:      ev.NewDueDate,
		ActorUserID:     ev.ActorUserID,
		CreatedAt:       ev.CreatedAt,
	}
	return mapFinanceErr(r.db.WithContext(ctx).Create(&m).Error)
}

func (r *FinanceEntryEventRepository) ListByEntry(ctx context.Context, workspaceID, entryID uuid.UUID) ([]dom.EntryEvent, error) {
	var rows []FinanceEntryEventModel
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND entry_id = ?", workspaceID, entryID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, mapFinanceErr(err)
	}
	toStatus := func(s *string) *dom.Status {
		if s == nil {
			return nil
		}
		v := dom.Status(*s)
		return &v
	}
	out := make([]dom.EntryEvent, len(rows))
	for i, m := range rows {
		out[i] = dom.EntryEvent{
			ID:              m.ID,
			WorkspaceID:     m.WorkspaceID,
			EntryID:         m.EntryID,
			Event:           dom.EntryEventType(m.Event),
			FromStatus:      toStatus(m.FromStatus),
			ToStatus:        toStatus(m.ToStatus),
			PaidAt:          m.PaidAt,
			PaidAmountCents: m.PaidAmountCents,
			CancelReason:    m.CancelReason,
			OldDueDate:      m.OldDueDate,
			NewDueDate:      m.NewDueDate,
			ActorUserID:     m.ActorUserID,
			CreatedAt:       m.CreatedAt,
		}
	}
	return out, nil
}
