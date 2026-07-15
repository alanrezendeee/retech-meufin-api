package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	appacc "github.com/retechfin/retechfin-api/internal/application/account"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserProfileModel mapeia a tabela user_profiles (1:1 com o usuário).
type UserProfileModel struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	WorkspaceID     uuid.UUID `gorm:"type:uuid;not null;column:workspace_id;index:idx_user_profiles_workspace"`
	AvatarObjectKey *string   `gorm:"column:avatar_object_key;size:500"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (UserProfileModel) TableName() string { return "user_profiles" }

// UserProfileRepository implementa account.UserProfileRepository sobre GORM.
type UserProfileRepository struct {
	db *gorm.DB
}

func NewUserProfileRepository(db *gorm.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

func (r *UserProfileRepository) Get(ctx context.Context, userID uuid.UUID) (*appacc.UserProfile, error) {
	var m UserProfileModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &appacc.UserProfile{
		UserID:          m.UserID,
		WorkspaceID:     m.WorkspaceID,
		AvatarObjectKey: m.AvatarObjectKey,
	}, nil
}

func (r *UserProfileRepository) Upsert(ctx context.Context, p *appacc.UserProfile) error {
	now := time.Now().UTC()
	m := UserProfileModel{
		UserID:          p.UserID,
		WorkspaceID:     p.WorkspaceID,
		AvatarObjectKey: p.AvatarObjectKey,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"workspace_id":      p.WorkspaceID,
				"avatar_object_key": p.AvatarObjectKey,
				"updated_at":        now,
			}),
		}).
		Create(&m).Error
}

func (r *UserProfileRepository) ClearAvatar(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&UserProfileModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"avatar_object_key": nil,
			"updated_at":        time.Now().UTC(),
		}).Error
}
