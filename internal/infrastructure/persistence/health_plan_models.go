package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthPlanModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID     uuid.UUID `gorm:"type:uuid;not null;index:idx_health_plans_workspace"`
	Name            string    `gorm:"size:255;not null"`
	Operator        *string   `gorm:"size:120"`
	PlanType        string    `gorm:"size:20;not null;default:familiar"`
	AnsCode         *string   `gorm:"column:ans_code;size:30"`
	MonthlyFeeCents int64     `gorm:"column:monthly_fee_cents;not null;default:0"`
	CoverageNotes   *string   `gorm:"type:text"`
	Active          bool      `gorm:"not null;default:true"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
	DeletedAt       gorm.DeletedAt
	Members         []HealthPlanMemberModel `gorm:"foreignKey:PlanID"`
}

func (HealthPlanModel) TableName() string { return "health_plans" }

type HealthPlanMemberModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index:idx_health_plan_members_workspace"`
	PlanID      uuid.UUID `gorm:"type:uuid;not null;index:idx_health_plan_members_unique"`
	MemberID    uuid.UUID `gorm:"type:uuid;not null;index:idx_health_plan_members_member"`
	CardNumber  *string   `gorm:"column:card_number;size:60"`
	Holder      bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (HealthPlanMemberModel) TableName() string { return "health_plan_members" }
