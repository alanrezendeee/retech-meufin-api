package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/homesafety"
	"gorm.io/gorm"
)

// HomeSafetyRepository implementa dom.Repository com GORM/Postgres.
type HomeSafetyRepository struct {
	db *gorm.DB
}

func NewHomeSafetyRepository(db *gorm.DB) *HomeSafetyRepository {
	return &HomeSafetyRepository{db: db}
}

// ─── Items ────────────────────────────────────────────────────────────────────

func (r *HomeSafetyRepository) CreateItem(ctx context.Context, i *dom.Item) error {
	m := homeSafetyItemToModel(i)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("home_safety: create item: %w", err)
	}
	return nil
}

func (r *HomeSafetyRepository) GetItemByID(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Item, error) {
	var m HomeSafetyItemModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item := modelToHomeSafetyItem(m)
	return &item, nil
}

func (r *HomeSafetyRepository) ListItems(ctx context.Context, workspaceID uuid.UUID, p dom.ListItemsParams) ([]dom.Item, error) {
	q := r.db.WithContext(ctx).Model(&HomeSafetyItemModel{}).
		Where("workspace_id = ?", workspaceID.String())
	if p.Category != "" {
		q = q.Where("category = ?", p.Category)
	}
	if p.Location != "" {
		q = q.Where("location = ?", p.Location)
	}
	if p.Query != "" {
		like := "%" + p.Query + "%"
		q = q.Where("name ILIKE ? OR brand ILIKE ? OR model ILIKE ?", like, like, like)
	}

	var models []HomeSafetyItemModel
	if err := q.Order("next_due_date ASC NULLS LAST, name ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Item, len(models))
	for i, m := range models {
		out[i] = modelToHomeSafetyItem(m)
	}
	return out, nil
}

func (r *HomeSafetyRepository) UpdateItem(ctx context.Context, i *dom.Item) error {
	m := homeSafetyItemToModel(i)
	if err := r.db.WithContext(ctx).Omit("CreatedAt").Save(&m).Error; err != nil {
		return fmt.Errorf("home_safety: update item: %w", err)
	}
	return nil
}

func (r *HomeSafetyRepository) DeleteItem(ctx context.Context, workspaceID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id.String(), workspaceID.String()).
		Delete(&HomeSafetyItemModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return dom.ErrNotFound
	}
	return nil
}

// ─── Events ───────────────────────────────────────────────────────────────────

func (r *HomeSafetyRepository) CreateEvent(ctx context.Context, e *dom.Event) error {
	m := homeSafetyEventToModel(e)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("home_safety: create event: %w", err)
	}
	return nil
}

func (r *HomeSafetyRepository) ListEvents(ctx context.Context, workspaceID, itemID uuid.UUID) ([]dom.Event, error) {
	var models []HomeSafetyEventModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND item_id = ?", workspaceID.String(), itemID.String()).
		Order("event_date DESC, created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Event, len(models))
	for i, m := range models {
		out[i] = modelToHomeSafetyEvent(m)
	}
	return out, nil
}

func (r *HomeSafetyRepository) DeleteEvent(ctx context.Context, workspaceID, itemID, eventID uuid.UUID) (*dom.Event, error) {
	var m HomeSafetyEventModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ? AND item_id = ?", eventID.String(), workspaceID.String(), itemID.String()).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Delete(&HomeSafetyEventModel{}, "id = ?", eventID.String()).Error; err != nil {
		return nil, err
	}
	ev := modelToHomeSafetyEvent(m)
	return &ev, nil
}

func (r *HomeSafetyRepository) ListMaintenanceCosts(ctx context.Context, workspaceID uuid.UUID) ([]dom.Event, error) {
	var models []HomeSafetyEventModel
	if err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND cost_cents > 0", workspaceID.String()).
		Order("event_date ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]dom.Event, len(models))
	for i, m := range models {
		out[i] = modelToHomeSafetyEvent(m)
	}
	return out, nil
}
