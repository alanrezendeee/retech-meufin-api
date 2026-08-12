package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthShareLinkModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID    uuid.UUID  `gorm:"type:uuid;not null"`
	Token          string     `gorm:"size:64;not null"`
	Scope          string     `gorm:"size:32;not null;default:member_panels"`
	FamilyMemberID uuid.UUID  `gorm:"type:uuid;not null"`
	Title          *string    `gorm:"size:255"`
	ExpiresAt      time.Time  `gorm:"not null"`
	ViewCount      int        `gorm:"not null;default:0"`
	LastViewedAt   *time.Time `gorm:"column:last_viewed_at"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	DeletedAt      gorm.DeletedAt
}

func (HealthShareLinkModel) TableName() string { return "health_share_links" }

type HealthShareLinkRepository struct{ db *gorm.DB }

func NewHealthShareLinkRepository(db *gorm.DB) *HealthShareLinkRepository {
	return &HealthShareLinkRepository{db: db}
}

func (r *HealthShareLinkRepository) Create(ctx context.Context, l *dom.ShareLink) error {
	m := shareLinkToModel(l)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return mapHealthErr(err)
	}
	return nil
}

func (r *HealthShareLinkRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]dom.ShareLink, error) {
	var models []HealthShareLinkModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.ShareLink, len(models))
	for i := range models {
		out[i] = *modelToShareLink(&models[i])
	}
	return out, nil
}

func (r *HealthShareLinkRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ShareLink, error) {
	var m HealthShareLinkModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dom.ErrNotFound
		}
		return nil, mapHealthErr(err)
	}
	return modelToShareLink(&m), nil
}

func (r *HealthShareLinkRepository) GetByToken(ctx context.Context, token string) (*dom.ShareLink, error) {
	var m HealthShareLinkModel
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dom.ErrNotFound
		}
		return nil, mapHealthErr(err)
	}
	return modelToShareLink(&m), nil
}

func (r *HealthShareLinkRepository) Update(ctx context.Context, l *dom.ShareLink) error {
	res := r.db.WithContext(ctx).Model(&HealthShareLinkModel{}).
		Where("id = ? AND workspace_id = ?", l.ID, l.WorkspaceID).
		Updates(map[string]any{
			"title":      l.Title,
			"revoked_at": l.RevokedAt,
			"updated_at": l.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthShareLinkRepository) CountActive(ctx context.Context, workspaceID uuid.UUID, now time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&HealthShareLinkModel{}).
		Where("workspace_id = ? AND revoked_at IS NULL AND expires_at > ?", workspaceID, now).
		Count(&n).Error
	if err != nil {
		return 0, mapHealthErr(err)
	}
	return n, nil
}

func (r *HealthShareLinkRepository) RegisterView(ctx context.Context, id uuid.UUID, when time.Time) error {
	return mapHealthErr(r.db.WithContext(ctx).Model(&HealthShareLinkModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"view_count":     gorm.Expr("view_count + 1"),
			"last_viewed_at": when,
			"updated_at":     when,
		}).Error)
}

func shareLinkToModel(l *dom.ShareLink) HealthShareLinkModel {
	return HealthShareLinkModel{
		ID:             l.ID,
		WorkspaceID:    l.WorkspaceID,
		Token:          l.Token,
		Scope:          l.Scope,
		FamilyMemberID: l.FamilyMemberID,
		Title:          l.Title,
		ExpiresAt:      l.ExpiresAt,
		ViewCount:      l.ViewCount,
		LastViewedAt:   l.LastViewedAt,
		CreatedBy:      l.CreatedBy,
		RevokedAt:      l.RevokedAt,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

func modelToShareLink(m *HealthShareLinkModel) *dom.ShareLink {
	return &dom.ShareLink{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		Token:          m.Token,
		Scope:          m.Scope,
		FamilyMemberID: m.FamilyMemberID,
		Title:          m.Title,
		ExpiresAt:      m.ExpiresAt,
		ViewCount:      m.ViewCount,
		LastViewedAt:   m.LastViewedAt,
		CreatedBy:      m.CreatedBy,
		RevokedAt:      m.RevokedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
