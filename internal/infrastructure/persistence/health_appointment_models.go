package persistence

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HealthAppointmentModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_appointments_workspace"`
	FamilyMemberID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_health_appointments_member"`
	Kind             string     `gorm:"size:20;not null;default:consulta"`
	Specialty        *string    `gorm:"size:30"`
	ProfessionalName *string    `gorm:"column:professional_name;size:255"`
	LabID            *uuid.UUID `gorm:"type:uuid"`
	ExamRequestID    *uuid.UUID `gorm:"column:exam_request_id;type:uuid"`
	PlanID           *uuid.UUID `gorm:"column:plan_id;type:uuid"`
	ScheduledAt      time.Time  `gorm:"column:scheduled_at;not null;index:idx_health_appointments_scheduled_at"`
	Status           string     `gorm:"size:20;not null;default:agendada;index:idx_health_appointments_status"`
	Reason           *string    `gorm:"type:text"`
	Outcome          *string    `gorm:"type:text"`
	PriceCents       int64      `gorm:"column:price_cents;not null;default:0"`
	CoveredByPlan    bool       `gorm:"column:covered_by_plan;not null;default:false"`
	Notes            *string    `gorm:"type:text"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
	DeletedAt        gorm.DeletedAt
}

func (HealthAppointmentModel) TableName() string { return "health_appointments" }
