// Package entitlement (application) resolve o plano/cota de um workspace e faz o
// metering mensal das consultas SEFAZ (pagas) via contador no Redis.
//
// Regra de negócio (DEC-0001): a cota conta apenas consultas SEFAZ bem-sucedidas.
// Reserva-se antes de consultar; em falha da SEFAZ devolve-se a reserva (refund),
// de modo que só o sucesso permanece debitado.
package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/entitlement"
	"github.com/retechfin/retechfin-api/internal/infrastructure/cache"
)

// counterTTL cobre com folga o mês corrente; a chave já contém o ano-mês, então
// meses anteriores expiram sozinhos.
const counterTTL = 35 * 24 * time.Hour

// Service resolve entitlements e faz o metering de consultas SEFAZ.
type Service struct {
	repo  dom.Repository
	cache *cache.Cache
}

func NewService(repo dom.Repository, c *cache.Cache) *Service {
	return &Service{repo: repo, cache: c}
}

// Resolve devolve o entitlement do workspace (ou o default 'free' quando não há
// registro explícito).
func (s *Service) Resolve(ctx context.Context, workspaceID uuid.UUID) (*dom.Entitlement, error) {
	e, err := s.repo.Get(ctx, workspaceID)
	if err == dom.ErrNotFound {
		return dom.Default(workspaceID), nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// FiscalUsage devolve o número de consultas SEFAZ já debitadas no mês corrente.
func (s *Service) FiscalUsage(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	n, err := s.cache.GetInt(ctx, s.fiscalKey(workspaceID))
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// FiscalStatus agrega tier, cota e uso do mês para exibição.
type FiscalStatus struct {
	Tier      dom.Tier
	Quota     int
	Used      int
	Remaining int
	Period    string // AAAA-MM do contador
}

// FiscalStatus resolve o panorama de cota SEFAZ do workspace.
func (s *Service) FiscalStatusFor(ctx context.Context, workspaceID uuid.UUID) (*FiscalStatus, error) {
	ent, err := s.Resolve(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	used, err := s.FiscalUsage(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	quota := ent.EffectiveFiscalSEFAZQuota()
	remaining := quota - used
	if remaining < 0 {
		remaining = 0
	}
	return &FiscalStatus{
		Tier:      ent.Tier,
		Quota:     quota,
		Used:      used,
		Remaining: remaining,
		Period:    time.Now().UTC().Format("2006-01"),
	}, nil
}

// ReserveFiscalSEFAZ reserva 1 consulta SEFAZ do mês. Retorna allowed=false
// quando a cota do mês já foi atingida (o chamador deve então degradar para IA).
// Sem Redis (cache nil), opera fail-open: allowed=true e sem metering.
func (s *Service) ReserveFiscalSEFAZ(ctx context.Context, workspaceID uuid.UUID) (allowed bool, remaining int, err error) {
	ent, err := s.Resolve(ctx, workspaceID)
	if err != nil {
		return false, 0, err
	}
	quota := ent.EffectiveFiscalSEFAZQuota()

	key := s.fiscalKey(workspaceID)
	n, err := s.cache.Incr(ctx, key, counterTTL)
	if err != nil {
		// Redis instável: não bloqueia o recurso (fail-open), mas não mete.
		return true, -1, nil
	}
	if n == 0 {
		// cache nil (sem Redis): fail-open, sem metering.
		return true, quota, nil
	}
	if int(n) > quota {
		_ = s.cache.Decr(ctx, key) // devolve a reserva que estourou a cota
		return false, 0, nil
	}
	return true, quota - int(n), nil
}

// RefundFiscalSEFAZ devolve uma reserva quando a consulta SEFAZ não se concretiza
// (falha/timeout) — só o sucesso deve permanecer debitado.
func (s *Service) RefundFiscalSEFAZ(ctx context.Context, workspaceID uuid.UUID) {
	_ = s.cache.Decr(ctx, s.fiscalKey(workspaceID))
}

func (s *Service) fiscalKey(workspaceID uuid.UUID) string {
	return fmt.Sprintf("usage:%s:fiscal_sefaz:%s", workspaceID.String(), time.Now().UTC().Format("200601"))
}
