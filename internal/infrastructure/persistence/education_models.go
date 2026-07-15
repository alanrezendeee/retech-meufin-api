package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/education"
)

// ─── GORM models ─────────────────────────────────────────────────────────────

type SchoolEnrollmentModel struct {
	ID                 string    `gorm:"primaryKey;column:id"`
	WorkspaceID        string    `gorm:"column:workspace_id"`
	MemberID           string    `gorm:"column:member_id"`
	SchoolYear         int       `gorm:"column:school_year"`
	Stage              string    `gorm:"column:stage;size:20"`
	SchoolName         *string   `gorm:"column:school_name;size:255"`
	Grade              *string   `gorm:"column:grade;size:60"`
	Shift              *string   `gorm:"column:shift;size:20"`
	MonthlyFeeCents    int64     `gorm:"column:monthly_fee_cents"`
	EnrollmentFeeCents int64     `gorm:"column:enrollment_fee_cents"`
	Notes              *string   `gorm:"column:notes"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SchoolEnrollmentModel) TableName() string { return "school_enrollments" }

type SchoolSupplyListModel struct {
	ID           string    `gorm:"primaryKey;column:id"`
	WorkspaceID  string    `gorm:"column:workspace_id"`
	EnrollmentID string    `gorm:"column:enrollment_id"`
	Title        string    `gorm:"column:title;size:255"`
	Status       string    `gorm:"column:status;size:20"`
	Notes        *string   `gorm:"column:notes"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	Items        []SchoolSupplyItemModel `gorm:"foreignKey:ListID;references:ID"`
}

func (SchoolSupplyListModel) TableName() string { return "school_supply_lists" }

type SchoolSupplyItemModel struct {
	ID                  string     `gorm:"primaryKey;column:id"`
	WorkspaceID         string     `gorm:"column:workspace_id"`
	ListID              string     `gorm:"column:list_id"`
	Name                string     `gorm:"column:name;size:255"`
	Category            string     `gorm:"column:category;size:20"`
	Quantity            float64    `gorm:"column:quantity"`
	ReferencePriceCents int64      `gorm:"column:reference_price_cents"`
	Purchased           bool       `gorm:"column:purchased"`
	PaidPriceCents      int64      `gorm:"column:paid_price_cents"`
	PurchasedAt         *time.Time `gorm:"column:purchased_at"`
	Store               *string    `gorm:"column:store;size:255"`
	Notes               *string    `gorm:"column:notes"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (SchoolSupplyItemModel) TableName() string { return "school_supply_items" }

// ─── Converters ────────────────────────────────────────────────────────────────

func enrollmentToModel(e *dom.SchoolEnrollment) SchoolEnrollmentModel {
	m := SchoolEnrollmentModel{
		ID:                 e.ID.String(),
		WorkspaceID:        e.WorkspaceID.String(),
		MemberID:           e.MemberID.String(),
		SchoolYear:         e.SchoolYear,
		Stage:              string(e.Stage),
		SchoolName:         e.SchoolName,
		Grade:              e.Grade,
		MonthlyFeeCents:    e.MonthlyFeeCents,
		EnrollmentFeeCents: e.EnrollmentFeeCents,
		Notes:              e.Notes,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
	if e.Shift != nil {
		s := string(*e.Shift)
		m.Shift = &s
	}
	return m
}

func modelToEnrollment(m SchoolEnrollmentModel) dom.SchoolEnrollment {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	memberID, _ := uuid.Parse(m.MemberID)
	e := dom.SchoolEnrollment{
		ID:                 id,
		WorkspaceID:        wsID,
		MemberID:           memberID,
		SchoolYear:         m.SchoolYear,
		Stage:              dom.Stage(m.Stage),
		SchoolName:         m.SchoolName,
		Grade:              m.Grade,
		MonthlyFeeCents:    m.MonthlyFeeCents,
		EnrollmentFeeCents: m.EnrollmentFeeCents,
		Notes:              m.Notes,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.Shift != nil {
		s := dom.Shift(*m.Shift)
		e.Shift = &s
	}
	return e
}

func listToModel(l *dom.SchoolSupplyList) SchoolSupplyListModel {
	return SchoolSupplyListModel{
		ID:           l.ID.String(),
		WorkspaceID:  l.WorkspaceID.String(),
		EnrollmentID: l.EnrollmentID.String(),
		Title:        l.Title,
		Status:       string(l.Status),
		Notes:        l.Notes,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func modelToList(m SchoolSupplyListModel) dom.SchoolSupplyList {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	enrollmentID, _ := uuid.Parse(m.EnrollmentID)
	l := dom.SchoolSupplyList{
		ID:           id,
		WorkspaceID:  wsID,
		EnrollmentID: enrollmentID,
		Title:        m.Title,
		Status:       dom.ListStatus(m.Status),
		Notes:        m.Notes,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	l.Items = make([]dom.SchoolSupplyItem, len(m.Items))
	for i, it := range m.Items {
		l.Items[i] = modelToItem(it)
	}
	return l
}

func itemToModel(i *dom.SchoolSupplyItem) SchoolSupplyItemModel {
	return SchoolSupplyItemModel{
		ID:                  i.ID.String(),
		WorkspaceID:         i.WorkspaceID.String(),
		ListID:              i.ListID.String(),
		Name:                i.Name,
		Category:            string(i.Category),
		Quantity:            i.Quantity,
		ReferencePriceCents: i.ReferencePriceCents,
		Purchased:           i.Purchased,
		PaidPriceCents:      i.PaidPriceCents,
		PurchasedAt:         i.PurchasedAt,
		Store:               i.Store,
		Notes:               i.Notes,
		CreatedAt:           i.CreatedAt,
		UpdatedAt:           i.UpdatedAt,
	}
}

func modelToItem(m SchoolSupplyItemModel) dom.SchoolSupplyItem {
	id, _ := uuid.Parse(m.ID)
	wsID, _ := uuid.Parse(m.WorkspaceID)
	listID, _ := uuid.Parse(m.ListID)
	return dom.SchoolSupplyItem{
		ID:                  id,
		WorkspaceID:         wsID,
		ListID:              listID,
		Name:                m.Name,
		Category:            dom.ItemCategory(m.Category),
		Quantity:            m.Quantity,
		ReferencePriceCents: m.ReferencePriceCents,
		Purchased:           m.Purchased,
		PaidPriceCents:      m.PaidPriceCents,
		PurchasedAt:         m.PurchasedAt,
		Store:               m.Store,
		Notes:               m.Notes,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}
