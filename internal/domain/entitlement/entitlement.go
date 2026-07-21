// Package entitlement modela o tier (plano) e as cotas de uso por workspace.
// Não há tabela de workspaces local: workspace_id é o UUID opaco do JWT, e a
// ausência de registro significa tier 'free' com as cotas padrão.
package entitlement

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound indica que o workspace não tem entitlement explícito (usa-se o default).
var ErrNotFound = errors.New("entitlement não encontrado")

// Tier é o plano do workspace. Define as cotas padrão.
type Tier string

const (
	TierFree Tier = "free"
	TierPro  Tier = "pro"
)

// ValidTier informa se o tier é conhecido.
func ValidTier(t Tier) bool {
	switch t {
	case TierFree, TierPro:
		return true
	default:
		return false
	}
}

// defaultFiscalSEFAZQuota é a cota mensal padrão de consultas SEFAZ (Infosimples)
// por tier. Valores a calibrar conforme uso real e franquia mínima do fornecedor.
var defaultFiscalSEFAZQuota = map[Tier]int{
	TierFree: 10,
	TierPro:  200,
}

// DefaultFiscalSEFAZQuota devolve a cota padrão do tier (0 para tier desconhecido).
func DefaultFiscalSEFAZQuota(t Tier) int {
	return defaultFiscalSEFAZQuota[t]
}

// Entitlement é o plano + cotas de um workspace.
type Entitlement struct {
	WorkspaceID uuid.UUID
	Tier        Tier
	// FiscalSEFAZQuota sobrescreve a cota padrão do tier quando != nil.
	FiscalSEFAZQuota *int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Default devolve o entitlement implícito (tier free) para um workspace sem registro.
func Default(workspaceID uuid.UUID) *Entitlement {
	return &Entitlement{WorkspaceID: workspaceID, Tier: TierFree}
}

// EffectiveFiscalSEFAZQuota resolve a cota mensal efetiva: override do workspace
// quando presente, senão o padrão do tier.
func (e *Entitlement) EffectiveFiscalSEFAZQuota() int {
	if e.FiscalSEFAZQuota != nil {
		return *e.FiscalSEFAZQuota
	}
	return DefaultFiscalSEFAZQuota(e.Tier)
}

// Repository persiste entitlements por workspace.
type Repository interface {
	// Get devolve o entitlement do workspace ou ErrNotFound quando não existe.
	Get(ctx context.Context, workspaceID uuid.UUID) (*Entitlement, error)
	// Upsert cria ou atualiza o entitlement do workspace.
	Upsert(ctx context.Context, e *Entitlement) error
}
