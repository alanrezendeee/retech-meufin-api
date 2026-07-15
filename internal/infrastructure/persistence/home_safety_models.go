package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/homesafety"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type HomeSafetyItemModel struct {
	ID                    string     `gorm:"primaryKey;column:id"`
	WorkspaceID           string     `gorm:"column:workspace_id"`
	Name                  string     `gorm:"column:name"`
	Category              string     `gorm:"column:category;size:30"`
	RiskType              string     `gorm:"column:risk_type;size:20"`
	Location              *string    `gorm:"column:location"`
	Brand                 *string    `gorm:"column:brand"`
	Model                 *string    `gorm:"column:model"`
	InstalledAt           *time.Time `gorm:"column:installed_at"`
	LifespanMonths        *int       `gorm:"column:lifespan_months"`
	ServiceIntervalMonths *int       `gorm:"column:service_interval_months"`
	LastServiceAt         *time.Time `gorm:"column:last_service_at"`
	NextDueDate           *time.Time `gorm:"column:next_due_date"`
	Priority              string     `gorm:"column:priority;size:10"`
	Responsible           *string    `gorm:"column:responsible"`
	LastCostCents         int64      `gorm:"column:last_cost_cents"`
	Active                bool       `gorm:"column:active"`
	Notes                 *string    `gorm:"column:notes"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (HomeSafetyItemModel) TableName() string { return "home_safety_items" }

type HomeSafetyEventModel struct {
	ID          string    `gorm:"primaryKey;column:id"`
	WorkspaceID string    `gorm:"column:workspace_id"`
	ItemID      string    `gorm:"column:item_id"`
	EventType   string    `gorm:"column:event_type;size:20"`
	EventDate   time.Time `gorm:"column:event_date"`
	CostCents   int64     `gorm:"column:cost_cents"`
	Provider    *string   `gorm:"column:provider"`
	Notes       *string   `gorm:"column:notes"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (HomeSafetyEventModel) TableName() string { return "home_safety_events" }

// ─── Converters ──────────────────────────────────────────────────────────────

func homeSafetyItemToModel(i *dom.Item) HomeSafetyItemModel {
	return HomeSafetyItemModel{
		ID:                    i.ID.String(),
		WorkspaceID:           i.WorkspaceID.String(),
		Name:                  i.Name,
		Category:              string(i.Category),
		RiskType:              string(i.RiskType),
		Location:              i.Location,
		Brand:                 i.Brand,
		Model:                 i.Model,
		InstalledAt:           i.InstalledAt,
		LifespanMonths:        i.LifespanMonths,
		ServiceIntervalMonths: i.ServiceIntervalMonths,
		LastServiceAt:         i.LastServiceAt,
		NextDueDate:           i.NextDueDate,
		Priority:              string(i.Priority),
		Responsible:           i.Responsible,
		LastCostCents:         i.LastCostCents,
		Active:                i.Active,
		Notes:                 i.Notes,
		CreatedAt:             i.CreatedAt,
		UpdatedAt:             i.UpdatedAt,
	}
}

func modelToHomeSafetyItem(m HomeSafetyItemModel) dom.Item {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	return dom.Item{
		ID:                    id,
		WorkspaceID:           wsID,
		Name:                  m.Name,
		Category:              dom.Category(m.Category),
		RiskType:              dom.RiskType(m.RiskType),
		Location:              m.Location,
		Brand:                 m.Brand,
		Model:                 m.Model,
		InstalledAt:           m.InstalledAt,
		LifespanMonths:        m.LifespanMonths,
		ServiceIntervalMonths: m.ServiceIntervalMonths,
		LastServiceAt:         m.LastServiceAt,
		NextDueDate:           m.NextDueDate,
		Priority:              dom.Priority(m.Priority),
		Responsible:           m.Responsible,
		LastCostCents:         m.LastCostCents,
		Active:                m.Active,
		Notes:                 m.Notes,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

func homeSafetyEventToModel(e *dom.Event) HomeSafetyEventModel {
	return HomeSafetyEventModel{
		ID:          e.ID.String(),
		WorkspaceID: e.WorkspaceID.String(),
		ItemID:      e.ItemID.String(),
		EventType:   string(e.EventType),
		EventDate:   e.EventDate,
		CostCents:   e.CostCents,
		Provider:    e.Provider,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
	}
}

func modelToHomeSafetyEvent(m HomeSafetyEventModel) dom.Event {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	itemID, _ := uuid.Parse(m.ItemID)
	return dom.Event{
		ID:          id,
		WorkspaceID: wsID,
		ItemID:      itemID,
		EventType:   dom.EventType(m.EventType),
		EventDate:   m.EventDate,
		CostCents:   m.CostCents,
		Provider:    m.Provider,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt,
	}
}
