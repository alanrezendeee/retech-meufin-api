package persistence

import (
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/entitlement"
)

// EntitlementModel mapeia a tabela workspace_entitlements.
type EntitlementModel struct {
	WorkspaceID      uuid.UUID `gorm:"type:uuid;primaryKey;column:workspace_id"`
	Tier             string    `gorm:"column:tier;size:30;not null;default:free"`
	FiscalSEFAZQuota *int      `gorm:"column:fiscal_sefaz_quota"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (EntitlementModel) TableName() string { return "workspace_entitlements" }

func entitlementToModel(e *dom.Entitlement) EntitlementModel {
	return EntitlementModel{
		WorkspaceID:      e.WorkspaceID,
		Tier:             string(e.Tier),
		FiscalSEFAZQuota: e.FiscalSEFAZQuota,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

func modelToEntitlement(m *EntitlementModel) dom.Entitlement {
	return dom.Entitlement{
		WorkspaceID:      m.WorkspaceID,
		Tier:             dom.Tier(m.Tier),
		FiscalSEFAZQuota: m.FiscalSEFAZQuota,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
