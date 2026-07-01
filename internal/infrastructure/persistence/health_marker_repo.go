package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthMarkerRepository struct {
	db *gorm.DB
}

func NewHealthMarkerRepository(db *gorm.DB) *HealthMarkerRepository {
	return &HealthMarkerRepository{db: db}
}

// scopeMarker filtra marcadores do sistema OU do próprio tenant.
func scopeMarker(db *gorm.DB, workspaceID uuid.UUID) *gorm.DB {
	return db.Where("(scope = ? OR (scope = ? AND workspace_id = ?))", "system", "tenant", workspaceID)
}

func (r *HealthMarkerRepository) Create(ctx context.Context, m *dom.Marker) error {
	model := markerToModel(m)
	model.Aliases = aliasesToModels(m.Aliases)
	return mapHealthErr(r.db.WithContext(ctx).Create(&model).Error)
}

func (r *HealthMarkerRepository) Update(ctx context.Context, m *dom.Marker) error {
	model := markerToModel(m)
	// Atualiza apenas o marcador (aliases não são editados neste MVP).
	res := r.db.WithContext(ctx).Model(&HealthMarkerModel{}).
		Where("id = ?", m.ID).
		Updates(map[string]any{
			"canonical_name":      model.CanonicalName,
			"normalized_key":      model.NormalizedKey,
			"loinc_code":          model.LoincCode,
			"category":            model.Category,
			"comparability_class": model.Comparability,
			"canonical_unit":      model.CanonicalUnit,
			"default_ref_min":     model.DefaultRefMin,
			"default_ref_max":     model.DefaultRefMax,
			"default_ref_text":    model.DefaultRefText,
			"active":              model.Active,
			"updated_at":          model.UpdatedAt,
		})
	if res.Error != nil {
		return mapHealthErr(res.Error)
	}
	if res.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

func (r *HealthMarkerRepository) SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND scope = ? AND workspace_id = ?", id, "tenant", workspaceID).
			Delete(&HealthMarkerModel{})
		if res.Error != nil {
			return mapHealthErr(res.Error)
		}
		if res.RowsAffected == 0 {
			return dom.ErrNotFound
		}
		// soft-delete dos aliases do marcador
		if err := tx.Where("marker_id = ?", id).Delete(&HealthMarkerAliasModel{}).Error; err != nil {
			return mapHealthErr(err)
		}
		return nil
	})
}

func (r *HealthMarkerRepository) GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Marker, error) {
	var m HealthMarkerModel
	err := scopeMarker(r.db.WithContext(ctx), workspaceID).
		Preload("Aliases").
		Where("id = ?", id).
		First(&m).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return modelToMarker(&m), nil
}

func (r *HealthMarkerRepository) Search(ctx context.Context, workspaceID uuid.UUID, query, category string, limit, offset int) ([]dom.Marker, int64, error) {
	base := scopeMarker(r.db.WithContext(ctx).Model(&HealthMarkerModel{}), workspaceID)
	if category != "" {
		base = base.Where("category = ?", category)
	}
	if q := dom.Normalize(query); q != "" {
		like := "%" + q + "%"
		base = base.Where(
			"normalized_key LIKE ? OR id IN (SELECT marker_id FROM health_marker_aliases WHERE normalized_alias LIKE ? AND deleted_at IS NULL)",
			like, like,
		)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	var rows []HealthMarkerModel
	if err := base.Preload("Aliases").Order("canonical_name ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, mapHealthErr(err)
	}
	out := make([]dom.Marker, len(rows))
	for i := range rows {
		out[i] = *modelToMarker(&rows[i])
	}
	return out, total, nil
}

func (r *HealthMarkerRepository) MatchExact(ctx context.Context, workspaceID uuid.UUID, normalized string) (*dom.Marker, error) {
	if normalized == "" {
		return nil, dom.ErrNotFound
	}
	// 1) por chave canônica
	var m HealthMarkerModel
	err := scopeMarker(r.db.WithContext(ctx), workspaceID).
		Preload("Aliases").
		Where("normalized_key = ?", normalized).
		First(&m).Error
	if err == nil {
		return modelToMarker(&m), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, mapHealthErr(err)
	}
	// 2) por alias
	var alias HealthMarkerAliasModel
	err = r.db.WithContext(ctx).
		Where("(scope = ? OR (scope = ? AND workspace_id = ?))", "system", "tenant", workspaceID).
		Where("normalized_alias = ?", normalized).
		First(&alias).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	return r.GetByID(ctx, workspaceID, alias.MarkerID)
}

func (r *HealthMarkerRepository) Candidates(ctx context.Context, workspaceID uuid.UUID, limit int) ([]dom.Marker, error) {
	var rows []HealthMarkerModel
	err := scopeMarker(r.db.WithContext(ctx), workspaceID).
		Preload("Aliases").
		Where("active = ?", true).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.Marker, len(rows))
	for i := range rows {
		out[i] = *modelToMarker(&rows[i])
	}
	return out, nil
}

func (r *HealthMarkerRepository) UpsertSystem(ctx context.Context, m *dom.Marker) (bool, error) {
	var existing HealthMarkerModel
	err := r.db.WithContext(ctx).
		Where("scope = ? AND normalized_key = ?", "system", m.NormalizedKey).
		First(&existing).Error
	if err == nil {
		return false, nil // já existe, idempotente
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, mapHealthErr(err)
	}
	model := markerToModel(m)
	model.Aliases = aliasesToModels(m.Aliases)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return false, mapHealthErr(err)
	}
	return true, nil
}

// --- conversões ---

func markerToModel(m *dom.Marker) HealthMarkerModel {
	return HealthMarkerModel{
		ID:             m.ID,
		Scope:          string(m.Scope),
		WorkspaceID:    m.WorkspaceID,
		CanonicalName:  m.CanonicalName,
		NormalizedKey:  m.NormalizedKey,
		LoincCode:      m.LoincCode,
		Category:       m.Category,
		Comparability:  string(m.Comparability),
		CanonicalUnit:  m.CanonicalUnit,
		DefaultRefMin:  m.DefaultRefMin,
		DefaultRefMax:  m.DefaultRefMax,
		DefaultRefText: m.DefaultRefText,
		Active:         m.Active,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func aliasesToModels(as []dom.MarkerAlias) []HealthMarkerAliasModel {
	out := make([]HealthMarkerAliasModel, len(as))
	for i := range as {
		out[i] = HealthMarkerAliasModel{
			ID:              as[i].ID,
			MarkerID:        as[i].MarkerID,
			Scope:           string(as[i].Scope),
			WorkspaceID:     as[i].WorkspaceID,
			Alias:           as[i].Alias,
			NormalizedAlias: as[i].NormalizedAlias,
			Source:          as[i].Source,
			CreatedAt:       as[i].CreatedAt,
			UpdatedAt:       as[i].UpdatedAt,
		}
	}
	return out
}

func modelToMarker(m *HealthMarkerModel) *dom.Marker {
	out := &dom.Marker{
		ID:             m.ID,
		Scope:          dom.Scope(m.Scope),
		WorkspaceID:    m.WorkspaceID,
		CanonicalName:  m.CanonicalName,
		NormalizedKey:  m.NormalizedKey,
		LoincCode:      m.LoincCode,
		Category:       m.Category,
		Comparability:  dom.ComparabilityClass(m.Comparability),
		CanonicalUnit:  m.CanonicalUnit,
		DefaultRefMin:  m.DefaultRefMin,
		DefaultRefMax:  m.DefaultRefMax,
		DefaultRefText: m.DefaultRefText,
		Active:         m.Active,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	for i := range m.Aliases {
		a := m.Aliases[i]
		out.Aliases = append(out.Aliases, dom.MarkerAlias{
			ID:              a.ID,
			MarkerID:        a.MarkerID,
			Scope:           dom.Scope(a.Scope),
			WorkspaceID:     a.WorkspaceID,
			Alias:           a.Alias,
			NormalizedAlias: a.NormalizedAlias,
			Source:          a.Source,
			CreatedAt:       a.CreatedAt,
			UpdatedAt:       a.UpdatedAt,
		})
	}
	return out
}
