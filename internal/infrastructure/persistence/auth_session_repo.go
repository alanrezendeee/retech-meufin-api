package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	domsess "github.com/retechfin/retechfin-api/internal/domain/session"
	"gorm.io/gorm"
)

// AuthSessionModel espelha auth_sessions. Tokens ficam cifrados (AES-GCM).
type AuthSessionModel struct {
	ID              string     `gorm:"size:64;primaryKey"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	Email           string     `gorm:"size:255;not null"`
	Name            string     `gorm:"size:255;not null"`
	TenantID        *uuid.UUID `gorm:"type:uuid"`
	AccessTokenEnc  []byte     `gorm:"not null"`
	RefreshTokenEnc []byte     `gorm:"not null"`
	AccessExpiresAt time.Time  `gorm:"not null"`
	ExpiresAt       time.Time  `gorm:"not null"`
	CreatedAt       time.Time  `gorm:"not null"`
	LastSeenAt      time.Time  `gorm:"not null"`
	RevokedAt       *time.Time
	IP              string `gorm:"size:64;not null"`
	UserAgent       string `gorm:"size:512;not null"`
}

func (AuthSessionModel) TableName() string { return "auth_sessions" }

// AuthSessionRepository implementa domsess.Store sobre GORM.
type AuthSessionRepository struct {
	db     *gorm.DB
	cipher *domsess.Cipher
}

func NewAuthSessionRepository(db *gorm.DB, cipher *domsess.Cipher) *AuthSessionRepository {
	return &AuthSessionRepository{db: db, cipher: cipher}
}

var _ domsess.Store = (*AuthSessionRepository)(nil)

func (r *AuthSessionRepository) seal(t domsess.Tokens) (access, refresh []byte, err error) {
	if access, err = r.cipher.Seal(t.Access); err != nil {
		return nil, nil, err
	}
	if refresh, err = r.cipher.Seal(t.Refresh); err != nil {
		return nil, nil, err
	}
	return access, refresh, nil
}

func (r *AuthSessionRepository) Create(ctx context.Context, s domsess.Session) error {
	access, refresh, err := r.seal(s.Tokens)
	if err != nil {
		return err
	}
	m := AuthSessionModel{
		ID: s.ID, UserID: s.User.ID, Email: s.User.Email, Name: s.User.Name, TenantID: s.TenantID,
		AccessTokenEnc: access, RefreshTokenEnc: refresh, AccessExpiresAt: s.Tokens.AccessExpiresAt,
		ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, LastSeenAt: s.LastSeenAt, RevokedAt: s.RevokedAt,
		IP: truncate(s.IP, 64), UserAgent: truncate(s.UserAgent, 512),
	}
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *AuthSessionRepository) Get(ctx context.Context, id string) (domsess.Session, error) {
	var m AuthSessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domsess.Session{}, domsess.ErrNotFound
		}
		return domsess.Session{}, err
	}
	access, err := r.cipher.Open(m.AccessTokenEnc)
	if err != nil {
		return domsess.Session{}, err
	}
	refresh, err := r.cipher.Open(m.RefreshTokenEnc)
	if err != nil {
		return domsess.Session{}, err
	}
	return domsess.Session{
		ID:         m.ID,
		User:       domsess.UserInfo{ID: m.UserID, Email: m.Email, Name: m.Name},
		TenantID:   m.TenantID,
		Tokens:     domsess.Tokens{Access: access, Refresh: refresh, AccessExpiresAt: m.AccessExpiresAt},
		ExpiresAt:  m.ExpiresAt,
		CreatedAt:  m.CreatedAt,
		LastSeenAt: m.LastSeenAt,
		RevokedAt:  m.RevokedAt,
		IP:         m.IP,
		UserAgent:  m.UserAgent,
	}, nil
}

func (r *AuthSessionRepository) Touch(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&AuthSessionModel{}).Where("id = ?", id).
		Update("last_seen_at", at).Error
}

func (r *AuthSessionRepository) UpdateTokens(ctx context.Context, id string, t domsess.Tokens) error {
	access, refresh, err := r.seal(t)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&AuthSessionModel{}).Where("id = ?", id).Updates(map[string]any{
		"access_token_enc":  access,
		"refresh_token_enc": refresh,
		"access_expires_at": t.AccessExpiresAt,
	}).Error
}

func (r *AuthSessionRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&AuthSessionModel{}).
		Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", at).Error
}

func (r *AuthSessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&AuthSessionModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", at)
	return res.RowsAffected, res.Error
}

func (r *AuthSessionRepository) PurgeExpired(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at < ?", before, before).
		Delete(&AuthSessionModel{})
	return res.RowsAffected, res.Error
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
